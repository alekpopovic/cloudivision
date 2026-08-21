package controller

import (
	"context"
	"errors"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/gitops"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReleaseReconcileAwaitsApproval(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	release.Spec.Approval.Required = true
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseAwaitingApproval {
		t.Fatalf("phase = %q, want AwaitingApproval", updated.Status.Phase)
	}
	if provider.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", provider.updateCalls)
	}
}

func TestReleaseReconcileUpdatesGitOpsOnce(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", provider.updateCalls)
	}
	if updated.Status.GitCommit != "abc123" {
		t.Fatalf("gitCommit = %q, want abc123", updated.Status.GitCommit)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeploying {
		t.Fatalf("phase = %q, want Deploying", updated.Status.Phase)
	}
}

func TestReleaseCommitCheckpointPreventsDuplicateAfterStatusFailure(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	failingClient := &statusFailureClient{Client: reconciler.Client, fail: true}
	reconciler.Client = failingClient

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err == nil {
		t.Fatal("first Reconcile() error = nil, want injected status failure")
	}
	checkpointed := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), checkpointed); err != nil {
		t.Fatalf("get checkpointed Release error = %v", err)
	}
	if checkpointed.Annotations[gitCommitCheckpointAnnotation] != "abc123" {
		t.Fatalf("checkpoint annotation = %q, want abc123", checkpointed.Annotations[gitCommitCheckpointAnnotation])
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls after failed status = %d, want 1", provider.updateCalls)
	}

	failingClient.fail = false
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls after retry = %d, want checkpoint reuse", provider.updateCalls)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get updated Release error = %v", err)
	}
	if updated.Status.GitCommit != "abc123" || updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeploying {
		t.Fatalf("status after retry = %#v", updated.Status)
	}
}

func TestReleaseTransientGitOpsFailureRequeues(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	provider.updateErr = errors.New("temporary network failure")

	result, err := reconciler.Reconcile(ctx, releaseRequestFor(release))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != externalStateRequeue {
		t.Fatalf("RequeueAfter = %s, want %s", result.RequeueAfter, externalStateRequeue)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeploying {
		t.Fatalf("phase = %q, want retryable Deploying", updated.Status.Phase)
	}
	if hasCondition(updated.Status.Conditions, domain.ConditionFailed) {
		t.Fatalf("conditions = %#v, transient error must not be terminal", updated.Status.Conditions)
	}
}

func TestReleaseReconcileAwaitsEnvironmentApproval(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	environment := &cicdv1alpha1.Environment{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.EnvironmentRef, Namespace: release.Namespace}, environment); err != nil {
		t.Fatalf("get Environment error = %v", err)
	}
	environment.Spec.RequiresApproval = true
	if err := reconciler.Update(ctx, environment); err != nil {
		t.Fatalf("update Environment error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseAwaitingApproval {
		t.Fatalf("phase = %q, want AwaitingApproval", updated.Status.Phase)
	}
	if provider.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", provider.updateCalls)
	}
}

func TestReleaseReconcileDeploysAfterApproval(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	environment := &cicdv1alpha1.Environment{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.EnvironmentRef, Namespace: release.Namespace}, environment); err != nil {
		t.Fatalf("get Environment error = %v", err)
	}
	environment.Spec.RequiresApproval = true
	if err := reconciler.Update(ctx, environment); err != nil {
		t.Fatalf("update Environment error = %v", err)
	}
	release.Spec.Approval.Required = true
	release.Spec.Approval.ApprovedBy = "alice"
	now := metav1.Now()
	release.Spec.Approval.ApprovedAt = &now
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}

	result, err := reconciler.Reconcile(ctx, releaseRequestFor(release))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != externalStateRequeue {
		t.Fatalf("RequeueAfter = %s, want %s", result.RequeueAfter, externalStateRequeue)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeploying {
		t.Fatalf("phase = %q, want Deploying", updated.Status.Phase)
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", provider.updateCalls)
	}
}

func TestReleaseReconcileRejectedReleaseDoesNotDeploy(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	release.Spec.Approval.Required = true
	release.Spec.Approval.RejectedBy = "alice"
	now := metav1.Now()
	release.Spec.Approval.RejectedAt = &now
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailed {
		t.Fatalf("phase = %q, want Failed", updated.Status.Phase)
	}
	if !hasConditionReason(updated.Status.Conditions, "Failed", "ReleaseRejected") {
		t.Fatalf("conditions = %#v, want ReleaseRejected failure", updated.Status.Conditions)
	}
	if provider.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", provider.updateCalls)
	}
}

func TestReleaseReconcileBlocksWhenSupplyChainPolicyIsNotSatisfied(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	environment := &cicdv1alpha1.Environment{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.EnvironmentRef, Namespace: release.Namespace}, environment); err != nil {
		t.Fatalf("get Environment error = %v", err)
	}
	environment.Spec.Type = cicdv1alpha1.EnvironmentTypeProduction
	environment.Spec.Policy = cicdv1alpha1.EnvironmentPolicySpec{
		RequireSignedImages: true,
		RequireSBOM:         true,
	}
	if err := reconciler.Update(ctx, environment); err != nil {
		t.Fatalf("update Environment error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailed {
		t.Fatalf("phase = %q, want Failed", updated.Status.Phase)
	}
	if !hasConditionReason(updated.Status.Conditions, "Failed", "PolicyNotSatisfied") {
		t.Fatalf("conditions = %#v, want PolicyNotSatisfied failure", updated.Status.Conditions)
	}
	if provider.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", provider.updateCalls)
	}
}

func TestReleaseReconcileAllowsSatisfiedSupplyChainPolicy(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	environment := &cicdv1alpha1.Environment{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.EnvironmentRef, Namespace: release.Namespace}, environment); err != nil {
		t.Fatalf("get Environment error = %v", err)
	}
	environment.Spec.Type = cicdv1alpha1.EnvironmentTypeProduction
	environment.Spec.Policy = cicdv1alpha1.EnvironmentPolicySpec{
		RequireSignedImages: true,
		RequireSBOM:         true,
	}
	if err := reconciler.Update(ctx, environment); err != nil {
		t.Fatalf("update Environment error = %v", err)
	}
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.BuildRunRef, Namespace: release.Namespace}, buildRun); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	buildRun.Status.SupplyChain = cicdv1alpha1.BuildRunSupplyChainStatus{
		SBOMDigest:   "sha256:sbom",
		SignatureRef: "cosign://signature",
	}
	if err := reconciler.Status().Update(ctx, buildRun); err != nil {
		t.Fatalf("update BuildRun status error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeploying {
		t.Fatalf("phase = %q, want Deploying", updated.Status.Phase)
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", provider.updateCalls)
	}
}

func TestReleaseReconcileMarksDeployedFromArgoCDStatus(t *testing.T) {
	ctx := context.Background()
	reconciler, release, _ := newReleaseReconciler(t)
	reconciler.StatusReader = &fakeGitOpsProvider{
		status: &gitops.DeploymentStatus{SyncStatus: "Synced", HealthStatus: "Healthy"},
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeployed {
		t.Fatalf("phase = %q, want Deployed", updated.Status.Phase)
	}
	if updated.Status.Deployment.SyncStatus != "Synced" || updated.Status.Deployment.HealthStatus != "Healthy" {
		t.Fatalf("deployment status = %#v", updated.Status.Deployment)
	}
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("terminal Reconcile() error = %v", err)
	}
}

func TestReleaseReconcileKeepsDeployingWhenArgoCDUnavailable(t *testing.T) {
	ctx := context.Background()
	reconciler, release, _ := newReleaseReconciler(t)
	reconciler.StatusReader = &fakeGitOpsProvider{statusErr: gitops.ErrDeploymentStatusUnavailable}

	result, err := reconciler.Reconcile(ctx, releaseRequestFor(release))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.RequeueAfter != externalStateRequeue {
		t.Fatalf("RequeueAfter = %s, want %s", result.RequeueAfter, externalStateRequeue)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseDeploying {
		t.Fatalf("phase = %q, want Deploying", updated.Status.Phase)
	}
	if !hasCondition(updated.Status.Conditions, "ArgoCDStatusUnavailable") {
		t.Fatalf("conditions = %#v, want ArgoCDStatusUnavailable", updated.Status.Conditions)
	}
}

func newReleaseReconciler(t *testing.T) (*ReleaseReconciler, *cicdv1alpha1.Release, *fakeGitOpsProvider) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}
	buildRun := testBuildRun()
	buildRun.Spec.GitOps.RepoURL = "https://github.com/cloudivision/gitops.git"
	buildRun.Spec.GitOps.Branch = "main"
	buildRun.Spec.GitOps.Path = "apps/sample"
	release := &cicdv1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-buildrun-sample-environment", Namespace: "ci"},
		Spec: cicdv1alpha1.ReleaseSpec{
			ProjectRef:     "sample-project",
			EnvironmentRef: "sample-environment",
			BuildRunRef:    buildRun.Name,
			Image: cicdv1alpha1.ImageRef{
				Repository: "ghcr.io/cloudivision/example",
				Tag:        "main",
			},
			Strategy: cicdv1alpha1.ReleaseStrategyGitOps,
		},
	}
	environment := &cicdv1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-environment", Namespace: "ci"},
		Spec: cicdv1alpha1.EnvironmentSpec{
			ProjectRef:       "sample-project",
			DisplayName:      "Sample",
			Namespace:        "sample",
			Type:             cicdv1alpha1.EnvironmentTypeDev,
			RequiresApproval: false,
			GitOps: cicdv1alpha1.EnvironmentGitOpsSpec{
				Provider:        cicdv1alpha1.GitOpsProviderArgoCD,
				ApplicationName: "sample",
				Namespace:       "argocd",
			},
		},
	}
	provider := &fakeGitOpsProvider{commit: "abc123", statusErr: gitops.ErrDeploymentStatusUnavailable}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.Release{}).
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}).
		WithObjects(buildRun, release, environment).
		Build()
	return &ReleaseReconciler{
		Client:         fakeClient,
		GitOpsProvider: provider,
		StatusReader:   provider,
	}, release, provider
}

type fakeGitOpsProvider struct {
	updateCalls int
	commit      string
	updateErr   error
	status      *gitops.DeploymentStatus
	statusErr   error
}

type statusFailureClient struct {
	client.Client
	fail bool
}

func (c *statusFailureClient) Status() client.SubResourceWriter {
	return statusFailureWriter{delegate: c.Client.Status(), parent: c}
}

type statusFailureWriter struct {
	delegate client.SubResourceWriter
	parent   *statusFailureClient
}

func (w statusFailureWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return w.delegate.Create(ctx, obj, subResource, opts...)
}

func (w statusFailureWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if w.parent.fail {
		return apiConflict(obj.GetName())
	}
	return w.delegate.Update(ctx, obj, opts...)
}

func (w statusFailureWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return w.delegate.Patch(ctx, obj, patch, opts...)
}

func apiConflict(name string) error {
	return apierrors.NewConflict(schema.GroupResource{Group: "cicd.cloudivision.io", Resource: "releases"}, name, errors.New("injected conflict"))
}

func (p *fakeGitOpsProvider) UpdateImage(context.Context, gitops.UpdateImageRequest) (*gitops.UpdateImageResult, error) {
	p.updateCalls++
	if p.updateErr != nil {
		return nil, p.updateErr
	}
	return &gitops.UpdateImageResult{Commit: p.commit}, nil
}

func (p fakeGitOpsProvider) ReadDeploymentStatus(context.Context, gitops.DeploymentStatusRequest) (*gitops.DeploymentStatus, error) {
	if p.statusErr != nil {
		return nil, p.statusErr
	}
	return p.status, nil
}

func releaseRequestFor(release *cicdv1alpha1.Release) ctrl.Request {
	return ctrl.Request{NamespacedName: releaseObjectKey(release)}
}

func releaseObjectKey(release *cicdv1alpha1.Release) types.NamespacedName {
	return types.NamespacedName{Name: release.Name, Namespace: release.Namespace}
}

func hasCondition(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

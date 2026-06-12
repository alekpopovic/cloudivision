package controller

import (
	"context"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/gitops"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
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
}

func TestReleaseReconcileKeepsDeployingWhenArgoCDUnavailable(t *testing.T) {
	ctx := context.Background()
	reconciler, release, _ := newReleaseReconciler(t)
	reconciler.StatusReader = &fakeGitOpsProvider{statusErr: gitops.ErrDeploymentStatusUnavailable}

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

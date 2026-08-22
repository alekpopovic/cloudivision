package controller

import (
	"context"
	"errors"
	"testing"
	"time"

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
	prProvider := &fakePullRequestProvider{status: "open"}
	reconciler.PullRequestProvider = prProvider

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
	if prProvider.createCalls != 0 || prProvider.readCalls != 0 {
		t.Fatalf("direct commit unexpectedly called PR provider: %#v", prProvider)
	}
	if updated.Status.GitCommit != "abc123" {
		t.Fatalf("gitCommit = %q, want abc123", updated.Status.GitCommit)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseWaitingForSync {
		t.Fatalf("phase = %q, want WaitingForSync", updated.Status.Phase)
	}
}

func TestReleasePullRequestPromotionCreatesOnePullRequest(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	release.Spec.PromotionMode = cicdv1alpha1.PromotionModePullRequest
	release.Spec.PullRequest = cicdv1alpha1.PullRequestSpec{TargetBranch: "production", Reviewers: []string{"platform"}, Labels: []string{"release"}}
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}
	prProvider := &fakePullRequestProvider{status: "open"}
	reconciler.PullRequestProvider = prProvider

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
	if prProvider.createCalls != 1 {
		t.Fatalf("createCalls = %d, want 1", prProvider.createCalls)
	}
	if provider.lastRequest.Branch != "cloudivision/"+release.Name || provider.lastRequest.BaseBranch != "production" {
		t.Fatalf("GitOps branches = head %q base %q", provider.lastRequest.Branch, provider.lastRequest.BaseBranch)
	}
	if updated.Status.PullRequest.URL == "" || updated.Status.PullRequest.MergeStatus != "open" {
		t.Fatalf("pullRequest status = %#v", updated.Status.PullRequest)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseGitOpsChangeCommitted {
		t.Fatalf("phase = %q, want GitOpsChangeCommitted while PR is open", updated.Status.Phase)
	}
}

func TestReleaseMergedPullRequestAllowsSyncWait(t *testing.T) {
	ctx := context.Background()
	reconciler, release, _ := newReleaseReconciler(t)
	release.Spec.PromotionMode = cicdv1alpha1.PromotionModePullRequest
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}
	prProvider := &fakePullRequestProvider{status: "open"}
	reconciler.PullRequestProvider = prProvider
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("create Reconcile() error = %v", err)
	}
	prProvider.status = "merged"
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("merge Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseWaitingForSync {
		t.Fatalf("phase = %q, want WaitingForSync", updated.Status.Phase)
	}
}

func TestReleaseClosedPullRequestBlocksPromotion(t *testing.T) {
	ctx := context.Background()
	reconciler, release, _ := newReleaseReconciler(t)
	release.Spec.PromotionMode = cicdv1alpha1.PromotionModePullRequest
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}
	prProvider := &fakePullRequestProvider{status: "closed"}
	reconciler.PullRequestProvider = prProvider
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("create Reconcile() error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("status Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedApproval {
		t.Fatalf("phase = %q, want FailedApproval", updated.Status.Phase)
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
	if updated.Status.GitCommit != "abc123" || updated.Status.Phase != cicdv1alpha1.ReleasePhaseGitOpsChangeCommitted {
		t.Fatalf("status after retry = %#v", updated.Status)
	}
}

func TestReleaseGitCommitFailureIsSpecific(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	provider.updateErr = errors.New("temporary network failure")

	_, err := reconciler.Reconcile(ctx, releaseRequestFor(release))
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedGitCommit {
		t.Fatalf("phase = %q, want FailedGitCommit", updated.Status.Phase)
	}
	if !hasCondition(updated.Status.Conditions, domain.ConditionFailed) {
		t.Fatalf("conditions = %#v, want terminal failure", updated.Status.Conditions)
	}
}

func TestReleaseGitOperationFailuresUseSpecificPhases(t *testing.T) {
	tests := []struct {
		name      string
		operation gitops.Operation
		phase     cicdv1alpha1.ReleasePhase
	}{
		{name: "clone", operation: gitops.OperationClone, phase: cicdv1alpha1.ReleasePhaseFailedGitClone},
		{name: "commit", operation: gitops.OperationCommit, phase: cicdv1alpha1.ReleasePhaseFailedGitCommit},
		{name: "push", operation: gitops.OperationPush, phase: cicdv1alpha1.ReleasePhaseFailedGitPush},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			reconciler, release, provider := newReleaseReconciler(t)
			provider.updateErr = &gitops.OperationError{Operation: test.operation, Err: errors.New("injected failure")}
			recorder := &countingEventRecorder{reasons: map[string]int{}}
			reconciler.Recorder = recorder

			if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
				t.Fatalf("Reconcile() error = %v", err)
			}
			updated := &cicdv1alpha1.Release{}
			if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
				t.Fatalf("get Release error = %v", err)
			}
			if updated.Status.Phase != test.phase {
				t.Fatalf("phase = %q, want %q", updated.Status.Phase, test.phase)
			}
			if updated.Status.Failure.Reason == "" || updated.Status.Failure.Message == "" {
				t.Fatalf("failure = %#v, want preserved reason and message", updated.Status.Failure)
			}
			if recorder.reasons[updated.Status.Failure.Reason] != 1 {
				t.Fatalf("events = %#v, want one failure event", recorder.reasons)
			}
		})
	}
}

func TestReleaseDeploymentTimeoutIsTerminal(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	release.Spec.DeploymentTimeout = metav1.Duration{Duration: time.Minute}
	if err := reconciler.Update(ctx, release); err != nil {
		t.Fatalf("update Release error = %v", err)
	}
	release.Status.StartedAt = &metav1.Time{Time: time.Now().Add(-2 * time.Minute)}
	release.Status.Phase = cicdv1alpha1.ReleasePhaseWaitingForSync
	if err := reconciler.Status().Update(ctx, release); err != nil {
		t.Fatalf("update Release status error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseTimedOut {
		t.Fatalf("phase = %q, want TimedOut", updated.Status.Phase)
	}
	if provider.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", provider.updateCalls)
	}
}

func TestReleaseProviderStatusFailureIsSpecific(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("commit Reconcile() error = %v", err)
	}
	provider.statusErr = errors.New("provider API denied request")
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("status Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedProviderStatus {
		t.Fatalf("phase = %q, want FailedProviderStatus", updated.Status.Phase)
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
	if !result.Requeue {
		t.Fatalf("result = %#v, want immediate requeue after commit", result)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseGitOpsChangeCommitted {
		t.Fatalf("phase = %q, want GitOpsChangeCommitted", updated.Status.Phase)
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", provider.updateCalls)
	}
	if updated.Status.Approval.ApprovedBy != "alice" || updated.Status.Approval.ApprovedAt == nil {
		t.Fatalf("approval status = %#v, want alice and timestamp", updated.Status.Approval)
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
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedApproval {
		t.Fatalf("phase = %q, want FailedApproval", updated.Status.Phase)
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
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedValidation {
		t.Fatalf("phase = %q, want FailedValidation", updated.Status.Phase)
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
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseGitOpsChangeCommitted {
		t.Fatalf("phase = %q, want GitOpsChangeCommitted", updated.Status.Phase)
	}
	if provider.updateCalls != 1 {
		t.Fatalf("updateCalls = %d, want 1", provider.updateCalls)
	}
}

func TestReleaseReconcileBlocksCriticalVulnerabilities(t *testing.T) {
	ctx := context.Background()
	reconciler, release, provider := newReleaseReconciler(t)
	environment := &cicdv1alpha1.Environment{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.EnvironmentRef, Namespace: release.Namespace}, environment); err != nil {
		t.Fatalf("get Environment error = %v", err)
	}
	environment.Spec.Policy.BlockCriticalVulnerabilities = true
	if err := reconciler.Update(ctx, environment); err != nil {
		t.Fatalf("update Environment error = %v", err)
	}
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: release.Spec.BuildRunRef, Namespace: release.Namespace}, buildRun); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	buildRun.Status.SupplyChain.ScannerResultsRef = "grype-results.json"
	buildRun.Status.SupplyChain.CriticalVulnerabilities = 2
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
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedValidation || updated.Status.Failure.Reason != "CriticalVulnerabilitiesFound" {
		t.Fatalf("status = %#v", updated.Status)
	}
	if provider.updateCalls != 0 {
		t.Fatalf("updateCalls = %d, want 0", provider.updateCalls)
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
	if _, err := reconciler.Reconcile(ctx, releaseRequestFor(release)); err != nil {
		t.Fatalf("status Reconcile() error = %v", err)
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
	if result.RequeueAfter != 0 || !result.Requeue {
		t.Fatalf("first result = %#v, want immediate requeue after commit", result)
	}
	result, err = reconciler.Reconcile(ctx, releaseRequestFor(release))
	if err != nil {
		t.Fatalf("status Reconcile() error = %v", err)
	}
	if result.RequeueAfter != externalStateRequeue {
		t.Fatalf("RequeueAfter = %s, want %s", result.RequeueAfter, externalStateRequeue)
	}

	updated := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, releaseObjectKey(release), updated); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseWaitingForSync {
		t.Fatalf("phase = %q, want WaitingForSync", updated.Status.Phase)
	}
	if !hasCondition(updated.Status.Conditions, "ProviderStatusUnavailable") {
		t.Fatalf("conditions = %#v, want ProviderStatusUnavailable", updated.Status.Conditions)
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
	lastRequest gitops.UpdateImageRequest
}

type fakePullRequestProvider struct {
	createCalls int
	readCalls   int
	status      string
}

type statusFailureClient struct {
	client.Client
	fail         bool
	passedUpdate bool
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
		if w.parent.passedUpdate {
			return apiConflict(obj.GetName())
		}
		w.parent.passedUpdate = true
	}
	return w.delegate.Update(ctx, obj, opts...)
}

func (w statusFailureWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return w.delegate.Patch(ctx, obj, patch, opts...)
}

func apiConflict(name string) error {
	return apierrors.NewConflict(schema.GroupResource{Group: "cicd.cloudivision.io", Resource: "releases"}, name, errors.New("injected conflict"))
}

func (p *fakeGitOpsProvider) UpdateImage(_ context.Context, req gitops.UpdateImageRequest) (*gitops.UpdateImageResult, error) {
	p.updateCalls++
	p.lastRequest = req
	if p.updateErr != nil {
		return nil, p.updateErr
	}
	return &gitops.UpdateImageResult{Commit: p.commit}, nil
}

func (p *fakePullRequestProvider) CreateOrUpdatePullRequest(_ context.Context, req gitops.PullRequestRequest) (*gitops.PullRequestResult, error) {
	p.createCalls++
	return &gitops.PullRequestResult{Provider: "github", URL: "https://github.com/example/repo/pull/1", Reference: "1", HeadBranch: req.HeadBranch, TargetBranch: req.TargetBranch, MergeStatus: p.status}, nil
}

func (p *fakePullRequestProvider) ReadPullRequestStatus(_ context.Context, req gitops.PullRequestStatusRequest) (*gitops.PullRequestResult, error) {
	p.readCalls++
	return &gitops.PullRequestResult{Provider: "github", URL: "https://github.com/example/repo/pull/1", Reference: req.Reference, HeadBranch: "cloudivision/release", TargetBranch: "main", MergeStatus: p.status}, nil
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

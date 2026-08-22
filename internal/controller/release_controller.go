package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/gitops"
	"github.com/cloudivision/cloudivision/internal/kube"
	"github.com/cloudivision/cloudivision/internal/observability"
	"github.com/cloudivision/cloudivision/internal/policy"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerconfig "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	gitCommitCheckpointAnnotation = "cloudivision.io/gitops-commit"
	externalStateRequeue          = 30 * time.Second
	defaultDeploymentTimeout      = 30 * time.Minute
)

// ReleaseReconciler reconciles Release resources.
type ReleaseReconciler struct {
	client.Client
	GitOpsProvider          gitops.Provider
	StatusReader            gitops.StatusReader
	PullRequestProvider     gitops.PullRequestProvider
	Recorder                EventRecorder
	PolicyEvaluator         policy.Evaluator
	MaxConcurrentReconciles int
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=releases/finalizers,verbs=update
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=environments,verbs=get;list;watch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=applications,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	started := time.Now()
	defer func() {
		observability.ObserveReconcile("release", started, err)
	}()
	return r.reconcile(ctx, req)
}

func (r *ReleaseReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("controller", "release", "namespace", req.Namespace, "release", req.Name)
	release := &cicdv1alpha1.Release{}
	if err := r.Get(ctx, req.NamespacedName, release); err != nil {
		if apierrors.IsNotFound(err) {
			observability.ClearReleaseInProgress(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get Release %s: %w", req.NamespacedName, err)
	}
	logger = logger.WithValues(
		"project", release.Spec.ProjectRef,
		"buildRun", release.Spec.BuildRunRef,
		"correlationId", release.Annotations[observability.CorrelationIDAnno],
	)
	if releasePhaseTerminal(release.Status.Phase) {
		return ctrl.Result{}, nil
	}

	if release.Spec.Approval.RejectedBy != "" {
		release.Status.Approval = approvalStatus(release)
		return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedApproval, "ReleaseRejected", fmt.Sprintf("Release was rejected by %s.", release.Spec.Approval.RejectedBy))
	}

	buildRun, err := r.loadBuildRun(ctx, release)
	if err != nil {
		return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedValidation, "BuildRunUnavailable", err.Error())
	}

	environment, err := r.loadEnvironment(ctx, release)
	if err != nil {
		return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedValidation, "EnvironmentUnavailable", err.Error())
	}

	policyEvaluator := r.PolicyEvaluator
	if policyEvaluator == nil {
		policyEvaluator = policy.NewDefaultEvaluator()
	}
	decision := policyEvaluator.EvaluateRelease(ctx, policy.ReleasePolicyInput{Release: release, BuildRun: buildRun, Environment: environment})
	release.Status.Policy = policy.ToStatus(decision)
	if releaseRequiresApproval(release, environment) && release.Spec.Approval.ApprovedBy == "" {
		return ctrl.Result{}, r.markAwaitingApproval(ctx, release)
	}
	if !decision.Allowed {
		return ctrl.Result{}, r.markPolicyDenied(ctx, release, decision)
	}
	release.Status.Approval = approvalStatus(release)
	if deploymentTimedOut(release, time.Now()) {
		return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseTimedOut, "DeploymentTimedOut", "Release exceeded its configured deployment timeout.")
	}

	commit := release.Status.GitCommit
	if commit == "" {
		commit = release.Annotations[gitCommitCheckpointAnnotation]
	}
	if commit == "" {
		if err := r.markPreparingGitOpsChange(ctx, release); err != nil {
			return ctrl.Result{}, err
		}
		provider := r.GitOpsProvider
		if provider == nil {
			provider = gitops.GitRepositoryProvider{}
		}
		branch, baseBranch := gitOpsBranches(release, buildRun)
		result, err := provider.UpdateImage(ctx, gitops.UpdateImageRequest{
			RepositoryURL: buildRun.Spec.GitOps.RepoURL,
			Branch:        branch,
			BaseBranch:    baseBranch,
			Path:          buildRun.Spec.GitOps.Path,
			Strategy:      buildRun.Spec.GitOps.Strategy,
			ReleaseName:   release.Name,
			Image:         release.Spec.Image,
		})
		if err != nil {
			phase, reason := gitFailure(err)
			return ctrl.Result{}, r.markFailed(ctx, release, phase, reason, err.Error())
		}
		if result == nil || result.Commit == "" {
			return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedGitCommit, "EmptyGitCommit", "GitOps provider returned an empty commit.")
		}
		commit = result.Commit
		if err := r.persistGitCommitCheckpoint(ctx, release, commit); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("updated GitOps repository", "commit", commit)
		release.Status.GitCommit = commit
		if releasePromotionMode(release) == cicdv1alpha1.PromotionModePullRequest {
			if err := r.ensurePullRequest(ctx, release, buildRun); err != nil {
				return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedProviderStatus, "PullRequestCreateFailed", err.Error())
			}
		}
		if err := r.markGitOpsChangeCommitted(ctx, release); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	release.Status.GitCommit = commit

	if release.Status.Phase == cicdv1alpha1.ReleasePhasePreparingGitOpsChange || release.Status.Phase == cicdv1alpha1.ReleasePhasePending || release.Status.Phase == "" {
		if releasePromotionMode(release) == cicdv1alpha1.PromotionModePullRequest && release.Status.PullRequest.Reference == "" {
			if err := r.ensurePullRequest(ctx, release, buildRun); err != nil {
				return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedProviderStatus, "PullRequestCreateFailed", err.Error())
			}
		}
		if err := r.markGitOpsChangeCommitted(ctx, release); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	if releasePromotionMode(release) == cicdv1alpha1.PromotionModePullRequest {
		ready, closed, err := r.syncPullRequestStatus(ctx, release, buildRun)
		if err != nil {
			return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedProviderStatus, "PullRequestStatusFailed", err.Error())
		}
		if closed {
			return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedApproval, "PullRequestClosed", "Pull request was closed without being merged.")
		}
		if !ready {
			return ctrl.Result{RequeueAfter: externalStateRequeue}, nil
		}
	}
	if err := r.markWaitingForSync(ctx, release); err != nil {
		return ctrl.Result{}, err
	}
	if environment.Spec.GitOps.Provider == cicdv1alpha1.GitOpsProviderGeneric {
		return ctrl.Result{}, r.markDeployed(ctx, release, "GitOpsCommitApplied", "Generic GitOps policy treats a committed change as deployed.")
	}
	return r.syncProviderStatus(ctx, release, environment)
}

func releasePhaseTerminal(phase cicdv1alpha1.ReleasePhase) bool {
	switch phase {
	case cicdv1alpha1.ReleasePhaseDeployed, cicdv1alpha1.ReleasePhaseRolledBack,
		cicdv1alpha1.ReleasePhaseFailedValidation, cicdv1alpha1.ReleasePhaseFailedApproval,
		cicdv1alpha1.ReleasePhaseFailedGitClone, cicdv1alpha1.ReleasePhaseFailedGitCommit,
		cicdv1alpha1.ReleasePhaseFailedGitPush, cicdv1alpha1.ReleasePhaseFailedProviderStatus,
		cicdv1alpha1.ReleasePhaseTimedOut:
		return true
	default:
		return false
	}
}

func releaseRequiresApproval(release *cicdv1alpha1.Release, environment *cicdv1alpha1.Environment) bool {
	return release.Spec.Approval.Required || (environment != nil && environment.Spec.RequiresApproval)
}

func releasePromotionMode(release *cicdv1alpha1.Release) cicdv1alpha1.PromotionMode {
	if release.Spec.PromotionMode == "" {
		return cicdv1alpha1.PromotionModeDirectCommit
	}
	return release.Spec.PromotionMode
}

func gitOpsBranches(release *cicdv1alpha1.Release, buildRun *cicdv1alpha1.BuildRun) (branch, baseBranch string) {
	if releasePromotionMode(release) != cicdv1alpha1.PromotionModePullRequest {
		return buildRun.Spec.GitOps.Branch, ""
	}
	target := release.Spec.PullRequest.TargetBranch
	if target == "" {
		target = buildRun.Spec.GitOps.Branch
	}
	if target == "" {
		target = "main"
	}
	return "cloudivision/" + release.Name, target
}

func (r *ReleaseReconciler) pullRequestProvider(repositoryURL string) gitops.PullRequestProvider {
	if r.PullRequestProvider != nil {
		return r.PullRequestProvider
	}
	return gitops.PullRequestProviderForRepository(repositoryURL)
}

func (r *ReleaseReconciler) ensurePullRequest(ctx context.Context, release *cicdv1alpha1.Release, buildRun *cicdv1alpha1.BuildRun) error {
	head, target := gitOpsBranches(release, buildRun)
	result, err := r.pullRequestProvider(buildRun.Spec.GitOps.RepoURL).CreateOrUpdatePullRequest(ctx, gitops.PullRequestRequest{
		RepositoryURL: buildRun.Spec.GitOps.RepoURL,
		ReleaseName:   release.Name,
		HeadBranch:    head,
		TargetBranch:  target,
		Title:         renderPromotionTemplate(release.Spec.PullRequest.TitleTemplate, "cloudivision: promote {{release.name}}", release),
		Body:          renderPromotionTemplate(release.Spec.PullRequest.BodyTemplate, "Promotes {{image}} for Release {{release.name}}.", release),
		Reviewers:     release.Spec.PullRequest.Reviewers,
		Labels:        release.Spec.PullRequest.Labels,
	})
	if err != nil {
		return err
	}
	if result == nil || result.Reference == "" {
		return errors.New("pull request provider returned an empty reference")
	}
	release.Status.PullRequest = pullRequestStatus(result)
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) syncPullRequestStatus(ctx context.Context, release *cicdv1alpha1.Release, buildRun *cicdv1alpha1.BuildRun) (ready, closed bool, err error) {
	if release.Status.PullRequest.Reference == "" {
		if err := r.ensurePullRequest(ctx, release, buildRun); err != nil {
			return false, false, err
		}
	}
	result, err := r.pullRequestProvider(buildRun.Spec.GitOps.RepoURL).ReadPullRequestStatus(ctx, gitops.PullRequestStatusRequest{
		RepositoryURL: buildRun.Spec.GitOps.RepoURL,
		Reference:     release.Status.PullRequest.Reference,
	})
	if err != nil {
		return false, false, err
	}
	if result == nil {
		return false, false, errors.New("pull request provider returned an empty status")
	}
	release.Status.PullRequest = pullRequestStatus(result)
	if err := r.updateReleaseStatus(ctx, release); err != nil {
		return false, false, err
	}
	switch strings.ToLower(result.MergeStatus) {
	case "merged":
		return true, false, nil
	case "closed":
		return false, true, nil
	default:
		return false, false, nil
	}
}

func pullRequestStatus(result *gitops.PullRequestResult) cicdv1alpha1.PullRequestStatus {
	return cicdv1alpha1.PullRequestStatus{
		Provider:     result.Provider,
		URL:          result.URL,
		Reference:    result.Reference,
		HeadBranch:   result.HeadBranch,
		TargetBranch: result.TargetBranch,
		MergeStatus:  result.MergeStatus,
	}
}

func renderPromotionTemplate(template, fallback string, release *cicdv1alpha1.Release) string {
	if template == "" {
		template = fallback
	}
	image := release.Spec.Image.Repository
	if release.Spec.Image.Digest != "" {
		image += "@" + release.Spec.Image.Digest
	} else if release.Spec.Image.Tag != "" {
		image += ":" + release.Spec.Image.Tag
	}
	replacer := strings.NewReplacer("{{release.name}}", release.Name, "{{image}}", image)
	return replacer.Replace(template)
}

func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &cicdv1alpha1.Release{}, ReleaseBuildRunIndex, func(obj client.Object) []string {
		return []string{obj.(*cicdv1alpha1.Release).Spec.BuildRunRef}
	}); err != nil {
		return fmt.Errorf("index Releases by BuildRun: %w", err)
	}
	if r.GitOpsProvider == nil {
		r.GitOpsProvider = gitops.GitRepositoryProvider{}
	}
	if r.StatusReader == nil {
		r.StatusReader = gitops.ArgoCDStatusReader{Client: mgr.GetClient()}
	}
	r.Recorder = mgr.GetEventRecorderFor("release-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&cicdv1alpha1.Release{}).
		WithOptions(controllerconfig.Options{MaxConcurrentReconciles: normalizedConcurrency(r.MaxConcurrentReconciles)}).
		Complete(r)
}

func (r *ReleaseReconciler) loadBuildRun(ctx context.Context, release *cicdv1alpha1.Release) (*cicdv1alpha1.BuildRun, error) {
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: release.Namespace, Name: release.Spec.BuildRunRef}, buildRun); err != nil {
		return nil, fmt.Errorf("load BuildRun %q: %w", release.Spec.BuildRunRef, err)
	}
	if !buildRun.Spec.GitOps.Enabled {
		return nil, errors.New("BuildRun GitOps is not enabled")
	}
	if buildRun.Spec.GitOps.RepoURL == "" {
		return nil, errors.New("BuildRun GitOps repository URL is required")
	}
	return buildRun, nil
}

func (r *ReleaseReconciler) loadEnvironment(ctx context.Context, release *cicdv1alpha1.Release) (*cicdv1alpha1.Environment, error) {
	environment := &cicdv1alpha1.Environment{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: release.Namespace, Name: release.Spec.EnvironmentRef}, environment); err != nil {
		return nil, fmt.Errorf("load Environment %q: %w", release.Spec.EnvironmentRef, err)
	}
	return environment, nil
}

func (r *ReleaseReconciler) markAwaitingApproval(ctx context.Context, release *cicdv1alpha1.Release) error {
	if release.Status.Phase == cicdv1alpha1.ReleasePhaseAwaitingApproval &&
		release.Status.ObservedGeneration == release.Generation &&
		conditionCurrent(release.Status.Conditions, "AwaitingApproval", release.Generation) {
		return nil
	}
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseAwaitingApproval
	release.Status.ObservedGeneration = release.Generation
	release.Status.Approval = approvalStatus(release)
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               "AwaitingApproval",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "ApprovalRequired",
		Message:            "Release requires approval before updating the GitOps repository.",
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markPolicyDenied(ctx context.Context, release *cicdv1alpha1.Release, decision policy.Decision) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseFailedValidation
	release.Status.ObservedGeneration = release.Generation
	release.Status.CompletedAt = &now
	release.Status.Failure = cicdv1alpha1.FailureStatus{Reason: decision.Reason, Message: decision.Message}
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type: "PolicyDenied", Status: metav1.ConditionTrue, ObservedGeneration: release.Generation,
		Reason: decision.Reason, Message: decision.Message, LastTransitionTime: now,
	})
	if r.Recorder != nil {
		r.Recorder.Event(release, "Warning", "PolicyDenied", decision.Message)
	}
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markPreparingGitOpsChange(ctx context.Context, release *cicdv1alpha1.Release) error {
	if release.Status.Phase == cicdv1alpha1.ReleasePhasePreparingGitOpsChange &&
		release.Status.ObservedGeneration == release.Generation &&
		conditionCurrent(release.Status.Conditions, "GitOpsPrepared", release.Generation) {
		return nil
	}
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhasePreparingGitOpsChange
	release.Status.ObservedGeneration = release.Generation
	if release.Status.StartedAt == nil {
		release.Status.StartedAt = &now
	}
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               "GitOpsPrepared",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "ConfigurationValidated",
		Message:            "Release image, environment, and GitOps configuration are valid.",
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markGitOpsChangeCommitted(ctx context.Context, release *cicdv1alpha1.Release) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseGitOpsChangeCommitted
	release.Status.ObservedGeneration = release.Generation
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               "GitOpsChangeCommitted",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "GitCommitPushed",
		Message:            "GitOps change was committed and pushed.",
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markWaitingForSync(ctx context.Context, release *cicdv1alpha1.Release) error {
	if release.Status.Phase == cicdv1alpha1.ReleasePhaseWaitingForSync {
		return nil
	}
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseWaitingForSync
	release.Status.ObservedGeneration = release.Generation
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               "DeploymentReady",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: release.Generation,
		Reason:             "WaitingForProviderSync",
		Message:            "GitOps change is waiting for provider sync and health confirmation.",
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markDeployed(ctx context.Context, release *cicdv1alpha1.Release, reason, message string) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	release.Status.CompletedAt = &now
	release.Status.ObservedGeneration = release.Generation
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
	observeReleaseDuration(release, now.Time)
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markFailed(ctx context.Context, release *cicdv1alpha1.Release, phase cicdv1alpha1.ReleasePhase, reason, message string) error {
	if release.Status.Phase == phase &&
		release.Status.ObservedGeneration == release.Generation &&
		hasConditionReason(release.Status.Conditions, domain.ConditionFailed, reason) {
		return nil
	}
	now := metav1.Now()
	release.Status.Phase = phase
	release.Status.ObservedGeneration = release.Generation
	release.Status.CompletedAt = &now
	release.Status.Failure = cicdv1alpha1.FailureStatus{Reason: reason, Message: message}
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
	if r.Recorder != nil {
		r.Recorder.Event(release, "Warning", reason, message)
	}
	observeGitOpsFailure(reason)
	observeReleaseDuration(release, now.Time)
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) persistGitCommitCheckpoint(ctx context.Context, release *cicdv1alpha1.Release, commit string) error {
	key := client.ObjectKeyFromObject(release)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &cicdv1alpha1.Release{}
		if err := r.Get(ctx, key, latest); err != nil {
			return err
		}
		annotations := latest.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}
		if existing := annotations[gitCommitCheckpointAnnotation]; existing != "" {
			if existing != commit {
				return fmt.Errorf("Release %s GitOps commit checkpoint is %q, refusing to replace it with %q", key, existing, commit)
			}
			release.Annotations = annotations
			return nil
		}
		annotations[gitCommitCheckpointAnnotation] = commit
		latest.SetAnnotations(annotations)
		if err := r.Update(ctx, latest); err != nil {
			return fmt.Errorf("persist GitOps commit checkpoint: %w", err)
		}
		release.Annotations = annotations
		return nil
	})
}

func (r *ReleaseReconciler) syncProviderStatus(ctx context.Context, release *cicdv1alpha1.Release, environment *cicdv1alpha1.Environment) (ctrl.Result, error) {
	reader := r.StatusReader
	if reader == nil {
		reader = gitops.ArgoCDStatusReader{Client: r.Client}
	}
	status, err := reader.ReadDeploymentStatus(ctx, gitops.DeploymentStatusRequest{
		Provider:        environment.Spec.GitOps.Provider,
		ApplicationName: environment.Spec.GitOps.ApplicationName,
		Namespace:       environment.Spec.GitOps.Namespace,
	})
	if err != nil {
		if errors.Is(err, gitops.ErrDeploymentStatusUnavailable) {
			domain.SetCondition(&release.Status.Conditions, metav1.Condition{
				Type:               "ProviderStatusUnavailable",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: release.Generation,
				Reason:             "ApplicationUnavailable",
				Message:            "GitOps provider status is not available; keeping release in WaitingForSync phase.",
			})
			if updateErr := r.updateReleaseStatus(ctx, release); updateErr != nil {
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{RequeueAfter: externalStateRequeue}, nil
		}
		return ctrl.Result{}, r.markFailed(ctx, release, cicdv1alpha1.ReleasePhaseFailedProviderStatus, "ProviderStatusFailed", err.Error())
	}
	release.Status.Deployment = cicdv1alpha1.ReleaseDeploymentStatus{
		Provider:        string(environment.Spec.GitOps.Provider),
		ApplicationName: environment.Spec.GitOps.ApplicationName,
		SyncStatus:      status.SyncStatus,
		HealthStatus:    status.HealthStatus,
	}
	if status.SyncStatus == "Synced" && status.HealthStatus == "Healthy" {
		return ctrl.Result{}, r.markDeployed(ctx, release, "ApplicationHealthy", "GitOps application is synced and healthy.")
	}
	if err := r.updateReleaseStatus(ctx, release); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: externalStateRequeue}, nil
}

func approvalStatus(release *cicdv1alpha1.Release) cicdv1alpha1.ReleaseApprovalStatus {
	return cicdv1alpha1.ReleaseApprovalStatus{
		ApprovedBy: release.Spec.Approval.ApprovedBy,
		ApprovedAt: release.Spec.Approval.ApprovedAt,
		RejectedBy: release.Spec.Approval.RejectedBy,
		RejectedAt: release.Spec.Approval.RejectedAt,
	}
}

func deploymentTimedOut(release *cicdv1alpha1.Release, now time.Time) bool {
	if release.Status.StartedAt == nil {
		return false
	}
	timeout := release.Spec.DeploymentTimeout.Duration
	if timeout <= 0 {
		timeout = defaultDeploymentTimeout
	}
	return !now.Before(release.Status.StartedAt.Add(timeout))
}

func gitFailure(err error) (cicdv1alpha1.ReleasePhase, string) {
	operationError := &gitops.OperationError{}
	if errors.As(err, &operationError) {
		switch operationError.Operation {
		case gitops.OperationClone:
			return cicdv1alpha1.ReleasePhaseFailedGitClone, "GitCloneFailed"
		case gitops.OperationPush:
			return cicdv1alpha1.ReleasePhaseFailedGitPush, "GitPushFailed"
		}
	}
	return cicdv1alpha1.ReleasePhaseFailedGitCommit, "GitCommitFailed"
}

func (r *ReleaseReconciler) updateReleaseStatus(ctx context.Context, release *cicdv1alpha1.Release) error {
	if err := kube.UpdateStatusWithRetry(ctx, r.Client, release); err != nil {
		return fmt.Errorf("update Release status: %w", err)
	}
	if release.Status.Phase != "" {
		observability.ReleaseTotal.WithLabelValues(string(release.Status.Phase)).Inc()
	}
	var startedAt *time.Time
	if release.Status.StartedAt != nil {
		started := release.Status.StartedAt.Time
		startedAt = &started
	}
	observability.ObserveReleaseInProgress(release.Namespace, release.Name, string(release.Status.Phase), startedAt)
	return nil
}

func observeReleaseDuration(release *cicdv1alpha1.Release, completed time.Time) {
	if release.Status.StartedAt != nil {
		observability.ReleaseDeploymentDuration.Observe(completed.Sub(release.Status.StartedAt.Time).Seconds())
	}
}

func observeGitOpsFailure(reason string) {
	operation := ""
	switch reason {
	case "GitCloneFailed":
		operation = "clone"
	case "GitCommitFailed", "EmptyGitCommit":
		operation = "commit"
	case "GitPushFailed":
		operation = "push"
	case "PullRequestCreateFailed", "PullRequestStatusFailed":
		operation = "pull_request"
	case "ProviderStatusFailed", "DeploymentStatusFailed":
		operation = "status"
	}
	if operation != "" {
		observability.GitOpsFailures.WithLabelValues(operation).Inc()
	}
}

func hasConditionReason(conditions []metav1.Condition, conditionType, reason string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Reason == reason && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

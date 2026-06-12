package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/gitops"
	"github.com/cloudivision/cloudivision/internal/observability"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReleaseReconciler reconciles Release resources.
type ReleaseReconciler struct {
	client.Client
	GitOpsProvider gitops.Provider
	StatusReader   gitops.StatusReader
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
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get Release %s: %w", req.NamespacedName, err)
	}
	logger = logger.WithValues(
		"project", release.Spec.ProjectRef,
		"buildRun", release.Spec.BuildRunRef,
		"correlationId", release.Annotations[observability.CorrelationIDAnno],
	)

	if release.Spec.Approval.Required && release.Spec.Approval.ApprovedBy == "" {
		return ctrl.Result{}, r.markAwaitingApproval(ctx, release)
	}

	buildRun, err := r.loadBuildRun(ctx, release)
	if err != nil {
		return ctrl.Result{}, r.markFailed(ctx, release, "BuildRunUnavailable", err.Error())
	}

	var environment *cicdv1alpha1.Environment
	if release.Spec.EnvironmentRef != "" {
		environment, err = r.loadEnvironment(ctx, release)
		if err != nil {
			return ctrl.Result{}, r.markFailed(ctx, release, "EnvironmentUnavailable", err.Error())
		}
		blocked, err := r.enforceEnvironmentPolicy(ctx, release, buildRun, environment)
		if blocked || err != nil {
			return ctrl.Result{}, err
		}
	}

	if release.Status.GitCommit == "" {
		provider := r.GitOpsProvider
		if provider == nil {
			provider = gitops.GitRepositoryProvider{}
		}
		result, err := provider.UpdateImage(ctx, gitops.UpdateImageRequest{
			RepositoryURL: buildRun.Spec.GitOps.RepoURL,
			Branch:        buildRun.Spec.GitOps.Branch,
			Path:          buildRun.Spec.GitOps.Path,
			Strategy:      buildRun.Spec.GitOps.Strategy,
			ReleaseName:   release.Name,
			Image:         release.Spec.Image,
		})
		if err != nil {
			return ctrl.Result{}, r.markFailed(ctx, release, "GitOpsUpdateFailed", err.Error())
		}
		release.Status.GitCommit = result.Commit
		logger.Info("updated GitOps repository", "commit", result.Commit)
	}

	if err := r.markDeploying(ctx, release); err != nil {
		return ctrl.Result{}, err
	}
	if environment != nil && environment.Spec.GitOps.Provider == cicdv1alpha1.GitOpsProviderArgoCD {
		return ctrl.Result{}, r.syncArgoCDStatus(ctx, release, environment)
	}
	return ctrl.Result{}, nil
}

func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.GitOpsProvider == nil {
		r.GitOpsProvider = gitops.GitRepositoryProvider{}
	}
	if r.StatusReader == nil {
		r.StatusReader = gitops.ArgoCDStatusReader{Client: mgr.GetClient()}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&cicdv1alpha1.Release{}).
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

func (r *ReleaseReconciler) enforceEnvironmentPolicy(ctx context.Context, release *cicdv1alpha1.Release, buildRun *cicdv1alpha1.BuildRun, environment *cicdv1alpha1.Environment) (bool, error) {
	policy := environment.Spec.Policy
	if policy.RequireSignedImages && buildRun.Status.SupplyChain.SignatureRef == "" {
		return true, r.markFailed(ctx, release, "PolicyNotSatisfied", fmt.Sprintf("Environment %q requires signed images, but BuildRun %q has no signature reference.", environment.Name, buildRun.Name))
	}
	if policy.RequireSBOM && buildRun.Status.SupplyChain.SBOMPath == "" && buildRun.Status.SupplyChain.SBOMDigest == "" {
		return true, r.markFailed(ctx, release, "PolicyNotSatisfied", fmt.Sprintf("Environment %q requires an SBOM, but BuildRun %q has no SBOM metadata.", environment.Name, buildRun.Name))
	}
	if policy.BlockCriticalVulnerabilities && buildRun.Status.SupplyChain.ScannerResultsRef == "" {
		return true, r.markFailed(ctx, release, "PolicyNotSatisfied", fmt.Sprintf("Environment %q blocks critical vulnerabilities, but BuildRun %q has no scanner results reference.", environment.Name, buildRun.Name))
	}
	return false, nil
}

func (r *ReleaseReconciler) markAwaitingApproval(ctx context.Context, release *cicdv1alpha1.Release) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseAwaitingApproval
	release.Status.ObservedGeneration = release.Generation
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

func (r *ReleaseReconciler) markDeploying(ctx context.Context, release *cicdv1alpha1.Release) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseDeploying
	release.Status.ObservedGeneration = release.Generation
	if release.Status.StartedAt == nil {
		release.Status.StartedAt = &now
	}
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               "GitOpsUpdated",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "GitCommitCreated",
		Message:            "GitOps repository has been updated.",
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markDeployed(ctx context.Context, release *cicdv1alpha1.Release) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	release.Status.CompletedAt = &now
	release.Status.ObservedGeneration = release.Generation
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "ApplicationHealthy",
		Message:            "GitOps application is synced and healthy.",
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) markFailed(ctx context.Context, release *cicdv1alpha1.Release, reason, message string) error {
	now := metav1.Now()
	release.Status.Phase = cicdv1alpha1.ReleasePhaseFailed
	release.Status.ObservedGeneration = release.Generation
	release.Status.CompletedAt = &now
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) syncArgoCDStatus(ctx context.Context, release *cicdv1alpha1.Release, environment *cicdv1alpha1.Environment) error {
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
				Type:               "ArgoCDStatusUnavailable",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: release.Generation,
				Reason:             "ApplicationUnavailable",
				Message:            "Argo CD Application status is not available; keeping release in Deploying phase.",
			})
			return r.updateReleaseStatus(ctx, release)
		}
		return r.markFailed(ctx, release, "ArgoCDStatusFailed", err.Error())
	}
	release.Status.Deployment = cicdv1alpha1.ReleaseDeploymentStatus{
		Provider:        string(environment.Spec.GitOps.Provider),
		ApplicationName: environment.Spec.GitOps.ApplicationName,
		SyncStatus:      status.SyncStatus,
		HealthStatus:    status.HealthStatus,
	}
	if status.SyncStatus == "Synced" && status.HealthStatus == "Healthy" {
		return r.markDeployed(ctx, release)
	}
	return r.updateReleaseStatus(ctx, release)
}

func (r *ReleaseReconciler) updateReleaseStatus(ctx context.Context, release *cicdv1alpha1.Release) error {
	if err := r.Status().Update(ctx, release); err != nil {
		return fmt.Errorf("update Release status: %w", err)
	}
	return nil
}

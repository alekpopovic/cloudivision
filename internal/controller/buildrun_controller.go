package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/executor"
	jobexecutor "github.com/cloudivision/cloudivision/internal/executor/job"
	tektonexecutor "github.com/cloudivision/cloudivision/internal/executor/tekton"
	"github.com/cloudivision/cloudivision/internal/observability"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	buildRunFinalizer = "cloudivision.io/buildrun-finalizer"
)

// BuildRunReconciler reconciles BuildRun resources.
type BuildRunReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Recorder  EventRecorder
	Executors map[cicdv1alpha1.ExecutorType]executor.PipelineExecutor
}

// EventRecorder is the subset of Kubernetes event recording used by the reconciler.
type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects;repositories;pipelinetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=releases,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *BuildRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	started := time.Now()
	defer func() {
		observability.ObserveReconcile("buildrun", started, err)
	}()
	return r.reconcile(ctx, req)
}

func (r *BuildRunReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("controller", "buildrun", "namespace", req.Namespace, "buildRun", req.Name)
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := r.Get(ctx, req.NamespacedName, buildRun); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get BuildRun %s: %w", req.NamespacedName, err)
	}
	logger = logger.WithValues(
		"project", buildRun.Spec.ProjectRef,
		"correlationId", buildRun.Annotations[observability.CorrelationIDAnno],
	)

	if !buildRun.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("removing BuildRun finalizer")
		return ctrl.Result{}, r.removeFinalizer(ctx, buildRun)
	}

	if !controllerutil.ContainsFinalizer(buildRun, buildRunFinalizer) {
		controllerutil.AddFinalizer(buildRun, buildRunFinalizer)
		if err := r.Update(ctx, buildRun); err != nil {
			return ctrl.Result{}, fmt.Errorf("add BuildRun finalizer: %w", err)
		}
		logger.Info("added BuildRun finalizer")
	}

	project, repository, template, err := r.loadReferences(ctx, buildRun)
	if err != nil {
		return ctrl.Result{}, r.markReferenceError(ctx, buildRun, err)
	}

	if isTerminalBuildRunPhase(buildRun.Status.Phase) {
		if buildRun.Status.Phase == cicdv1alpha1.BuildRunPhaseSucceeded {
			return ctrl.Result{}, r.ensureRelease(ctx, buildRun)
		}
		return ctrl.Result{}, nil
	}

	pipelineExecutor, executorType, err := r.executorFor(buildRun)
	if err != nil {
		return ctrl.Result{}, r.markExecutorError(ctx, buildRun, err)
	}
	runRef, err := pipelineExecutor.EnsureRun(ctx, executor.EnsureRunRequest{
		BuildRun:   buildRun,
		Project:    project,
		Repository: repository,
		Template:   template,
	})
	if err != nil {
		return ctrl.Result{}, r.markExecutorError(ctx, buildRun, err)
	}
	if buildRun.Status.Phase == "" || buildRun.Status.Phase == cicdv1alpha1.BuildRunPhasePending {
		logger.Info("pipeline run ensured", "executor", executorType, "runKind", runRef.Kind, "runName", runRef.Name)
		r.recordRunCreated(buildRun, executorType, runRef)
		return ctrl.Result{}, r.markQueued(ctx, buildRun, *runRef)
	}
	runStatus, err := pipelineExecutor.ReadRunStatus(ctx, *runRef)
	if err != nil {
		return ctrl.Result{}, r.markExecutorError(ctx, buildRun, err)
	}
	return ctrl.Result{}, r.syncStatusFromRun(ctx, buildRun, *runRef, runStatus)
}

func (r *BuildRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Scheme = mgr.GetScheme()
	r.Recorder = mgr.GetEventRecorderFor("buildrun-controller")
	r.ensureDefaultExecutors()
	return ctrl.NewControllerManagedBy(mgr).
		For(&cicdv1alpha1.BuildRun{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

func (r *BuildRunReconciler) removeFinalizer(ctx context.Context, buildRun *cicdv1alpha1.BuildRun) error {
	if !controllerutil.ContainsFinalizer(buildRun, buildRunFinalizer) {
		return nil
	}
	controllerutil.RemoveFinalizer(buildRun, buildRunFinalizer)
	if err := r.Update(ctx, buildRun); err != nil {
		return fmt.Errorf("remove BuildRun finalizer: %w", err)
	}
	return nil
}

func (r *BuildRunReconciler) loadReferences(ctx context.Context, buildRun *cicdv1alpha1.BuildRun) (*cicdv1alpha1.Project, *cicdv1alpha1.Repository, *cicdv1alpha1.PipelineTemplate, error) {
	project := &cicdv1alpha1.Project{}
	if err := r.Get(ctx, types.NamespacedName{Name: buildRun.Spec.ProjectRef, Namespace: buildRun.Namespace}, project); err != nil {
		return nil, nil, nil, fmt.Errorf("load Project %q: %w", buildRun.Spec.ProjectRef, err)
	}
	if project.Spec.Namespace != "" && project.Spec.Namespace != buildRun.Namespace {
		return nil, nil, nil, fmt.Errorf("Project %q spec.namespace %q must match BuildRun namespace %q so runner ownerReferences and namespaced RBAC remain valid", project.Name, project.Spec.Namespace, buildRun.Namespace)
	}
	repository := &cicdv1alpha1.Repository{}
	if err := r.Get(ctx, types.NamespacedName{Name: buildRun.Spec.RepositoryRef, Namespace: buildRun.Namespace}, repository); err != nil {
		return nil, nil, nil, fmt.Errorf("load Repository %q: %w", buildRun.Spec.RepositoryRef, err)
	}
	template := &cicdv1alpha1.PipelineTemplate{}
	if err := r.Get(ctx, types.NamespacedName{Name: buildRun.Spec.PipelineTemplateRef, Namespace: buildRun.Namespace}, template); err != nil {
		return nil, nil, nil, fmt.Errorf("load PipelineTemplate %q: %w", buildRun.Spec.PipelineTemplateRef, err)
	}
	return project, repository, template, nil
}

func (r *BuildRunReconciler) markReferenceError(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, err error) error {
	now := metav1.Now()
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseFailed
	buildRun.Status.ObservedGeneration = buildRun.Generation
	buildRun.Status.CompletedAt = &now
	buildRun.Status.Failure = cicdv1alpha1.FailureStatus{
		Reason:  "InvalidReference",
		Message: err.Error(),
	}
	domain.SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: buildRun.Generation,
		Reason:             "InvalidReference",
		Message:            err.Error(),
		LastTransitionTime: now,
	})
	r.record(buildRun, corev1.EventTypeWarning, "BuildFailed", err.Error())
	return r.updateBuildRunStatus(ctx, buildRun)
}

func (r *BuildRunReconciler) markExecutorError(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, err error) error {
	reason := "ExecutorFailed"
	if errors.Is(err, tektonexecutor.ErrTektonUnavailable) {
		reason = "TektonUnavailable"
	}
	now := metav1.Now()
	if markErr := domain.MarkBuildRunFailed(buildRun, now, reason, err.Error()); markErr != nil {
		return markErr
	}
	r.record(buildRun, corev1.EventTypeWarning, "BuildFailed", err.Error())
	return r.updateBuildRunStatus(ctx, buildRun)
}

func (r *BuildRunReconciler) markQueued(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, ref executor.RunRef) error {
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseQueued
	buildRun.Status.ObservedGeneration = buildRun.Generation
	setRunRefStatus(buildRun, ref)
	domain.SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               "Queued",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: buildRun.Generation,
		Reason:             ref.Kind + "Created",
		Message:            "Pipeline run has been created.",
	})
	return r.updateBuildRunStatus(ctx, buildRun)
}

func (r *BuildRunReconciler) syncStatusFromRun(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, ref executor.RunRef, status *executor.RunStatus) error {
	setRunRefStatus(buildRun, ref)
	now := metav1.Now()

	switch status.Phase {
	case executor.RunPhaseSucceeded:
		if err := domain.MarkBuildRunSucceeded(buildRun, now, releaseImage(buildRun)); err != nil {
			return err
		}
		r.record(buildRun, corev1.EventTypeNormal, "BuildSucceeded", "Pipeline run completed successfully")
		if err := r.updateBuildRunStatus(ctx, buildRun); err != nil {
			return err
		}
		return r.ensureRelease(ctx, buildRun)
	case executor.RunPhaseFailed:
		if err := domain.MarkBuildRunFailed(buildRun, now, status.Failure.Reason, status.Failure.Message); err != nil {
			return err
		}
		r.record(buildRun, corev1.EventTypeWarning, "BuildFailed", status.Failure.Message)
		return r.updateBuildRunStatus(ctx, buildRun)
	case executor.RunPhaseRunning:
		if err := domain.MarkBuildRunStarted(buildRun, now); err != nil {
			return err
		}
		r.record(buildRun, corev1.EventTypeNormal, "BuildStarted", "Pipeline run is running")
		return r.updateBuildRunStatus(ctx, buildRun)
	}

	if buildRun.Status.Phase == "" {
		buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseQueued
		buildRun.Status.ObservedGeneration = buildRun.Generation
		return r.updateBuildRunStatus(ctx, buildRun)
	}
	return nil
}

func (r *BuildRunReconciler) executorFor(buildRun *cicdv1alpha1.BuildRun) (executor.PipelineExecutor, cicdv1alpha1.ExecutorType, error) {
	r.ensureDefaultExecutors()
	executorType := buildRun.Spec.Executor
	if executorType == "" {
		executorType = cicdv1alpha1.ExecutorTypeJob
	}
	if executorType == cicdv1alpha1.ExecutorTypeTekton && !tektonEnabled() {
		return nil, executorType, fmt.Errorf("%w: set CLOU_DIVISION_ENABLE_TEKTON=true to use executor=tekton", tektonexecutor.ErrTektonUnavailable)
	}
	pipelineExecutor := r.Executors[executorType]
	if pipelineExecutor == nil {
		return nil, executorType, fmt.Errorf("unsupported BuildRun executor %q", executorType)
	}
	return pipelineExecutor, executorType, nil
}

func (r *BuildRunReconciler) ensureDefaultExecutors() {
	if r.Executors == nil {
		r.Executors = map[cicdv1alpha1.ExecutorType]executor.PipelineExecutor{}
	}
	if r.Executors[cicdv1alpha1.ExecutorTypeJob] == nil {
		r.Executors[cicdv1alpha1.ExecutorTypeJob] = jobexecutor.Executor{Client: r.Client, Scheme: r.Scheme}
	}
	if r.Executors[cicdv1alpha1.ExecutorTypeTekton] == nil {
		r.Executors[cicdv1alpha1.ExecutorTypeTekton] = tektonexecutor.Executor{Client: r.Client, Scheme: r.Scheme}
	}
}

func (r *BuildRunReconciler) updateBuildRunStatus(ctx context.Context, buildRun *cicdv1alpha1.BuildRun) error {
	if err := r.Status().Update(ctx, buildRun); err != nil {
		return fmt.Errorf("update BuildRun status: %w", err)
	}
	if buildRun.Status.Phase != "" {
		observability.BuildRunTotal.WithLabelValues(string(buildRun.Status.Phase)).Inc()
	}
	if buildRun.Status.StartedAt != nil && buildRun.Status.CompletedAt != nil {
		observability.BuildRunDuration.Observe(buildRun.Status.CompletedAt.Sub(buildRun.Status.StartedAt.Time).Seconds())
	}
	return nil
}

func (r *BuildRunReconciler) ensureRelease(ctx context.Context, buildRun *cicdv1alpha1.BuildRun) error {
	if !buildRun.Spec.GitOps.Enabled {
		return nil
	}
	name := releaseNameForBuildRun(buildRun.Name, buildRun.Spec.GitOps.EnvironmentRef)
	release := &cicdv1alpha1.Release{}
	key := types.NamespacedName{Name: name, Namespace: buildRun.Namespace}
	if err := r.Get(ctx, key, release); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("get Release %s: %w", key, err)
		}
		release = &cicdv1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: buildRun.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":   "cloudivision",
					"cloudivision.io/buildrun": buildRun.Name,
					"cloudivision.io/project":  buildRun.Spec.ProjectRef,
				},
			},
			Spec: cicdv1alpha1.ReleaseSpec{
				ProjectRef:     buildRun.Spec.ProjectRef,
				BuildRunRef:    buildRun.Name,
				EnvironmentRef: buildRun.Spec.GitOps.EnvironmentRef,
				Image:          releaseImage(buildRun),
				Strategy:       cicdv1alpha1.ReleaseStrategyGitOps,
			},
		}
		if err := controllerutil.SetControllerReference(buildRun, release, r.Scheme); err != nil {
			return fmt.Errorf("set Release owner reference: %w", err)
		}
		if err := r.Create(ctx, release); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create Release %s: %w", key, err)
		}
		return nil
	}
	return nil
}

func (r *BuildRunReconciler) record(buildRun *cicdv1alpha1.BuildRun, eventType, reason, message string) {
	if r.Recorder != nil {
		r.Recorder.Event(buildRun, eventType, reason, message)
	}
}

func releaseNameForBuildRun(buildRunName, environment string) string {
	if environment == "" {
		environment = "default"
	}
	name := strings.ToLower(buildRunName + "-" + environment)
	name = strings.NewReplacer("_", "-", ".", "-").Replace(name)
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

func releaseImage(buildRun *cicdv1alpha1.BuildRun) cicdv1alpha1.ImageRef {
	if buildRun.Status.Image.Repository != "" {
		return buildRun.Status.Image
	}
	return buildRun.Spec.Image
}

func isTerminalBuildRunPhase(phase cicdv1alpha1.BuildRunPhase) bool {
	return phase == cicdv1alpha1.BuildRunPhaseSucceeded ||
		phase == cicdv1alpha1.BuildRunPhaseFailed ||
		phase == cicdv1alpha1.BuildRunPhaseCancelled
}

func setRunRefStatus(buildRun *cicdv1alpha1.BuildRun, ref executor.RunRef) {
	switch ref.Kind {
	case "PipelineRun":
		buildRun.Status.PipelineRunRef = cicdv1alpha1.ObjectRef{Name: ref.Name, Namespace: ref.Namespace}
	default:
		buildRun.Status.JobRef = cicdv1alpha1.ObjectRef{Name: ref.Name, Namespace: ref.Namespace}
	}
}

func (r *BuildRunReconciler) recordRunCreated(buildRun *cicdv1alpha1.BuildRun, executorType cicdv1alpha1.ExecutorType, ref *executor.RunRef) {
	if ref == nil {
		return
	}
	reason := ref.Kind + "Created"
	message := "Created pipeline run " + ref.Name
	if executorType == cicdv1alpha1.ExecutorTypeJob {
		reason = "JobCreated"
		message = "Created runner Job " + ref.Name
	}
	r.record(buildRun, corev1.EventTypeNormal, reason, message)
}

func tektonEnabled() bool {
	return strings.EqualFold(os.Getenv("CLOU_DIVISION_ENABLE_TEKTON"), "true")
}

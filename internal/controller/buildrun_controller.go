package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	buildRunFinalizer                    = "cloudivision.io/buildrun-finalizer"
	defaultRunnerImage                   = "cloudivision/runner:dev"
	defaultTTLSecondsAfterFinished int32 = 3600
	defaultBackoffLimit            int32 = 0
)

// BuildRunReconciler reconciles BuildRun resources.
type BuildRunReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder EventRecorder
}

// EventRecorder is the subset of Kubernetes event recording used by the reconciler.
type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects;repositories;pipelinetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *BuildRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("buildRun", req.NamespacedName)

	buildRun := &cicdv1alpha1.BuildRun{}
	if err := r.Get(ctx, req.NamespacedName, buildRun); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get BuildRun %s: %w", req.NamespacedName, err)
	}

	if !buildRun.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.removeFinalizer(ctx, buildRun)
	}

	if !controllerutil.ContainsFinalizer(buildRun, buildRunFinalizer) {
		controllerutil.AddFinalizer(buildRun, buildRunFinalizer)
		if err := r.Update(ctx, buildRun); err != nil {
			return ctrl.Result{}, fmt.Errorf("add BuildRun finalizer: %w", err)
		}
	}

	project, repository, template, err := r.loadReferences(ctx, buildRun)
	if err != nil {
		return ctrl.Result{}, r.markReferenceError(ctx, buildRun, err)
	}

	if buildRun.Spec.Executor != "" && buildRun.Spec.Executor != cicdv1alpha1.ExecutorTypeJob {
		logger.V(1).Info("BuildRun executor is not handled by Job reconciler", "executor", buildRun.Spec.Executor)
		return ctrl.Result{}, nil
	}

	if isTerminalBuildRunPhase(buildRun.Status.Phase) {
		return ctrl.Result{}, nil
	}

	job := &batchv1.Job{}
	jobKey := types.NamespacedName{
		Name:      jobNameForBuildRun(buildRun.Name),
		Namespace: buildRun.Namespace,
	}
	if err := r.Get(ctx, jobKey, job); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("get Job %s: %w", jobKey, err)
		}
		job = r.buildJob(buildRun, project, repository, template)
		if err := controllerutil.SetControllerReference(buildRun, job, r.Scheme); err != nil {
			return ctrl.Result{}, fmt.Errorf("set Job owner reference: %w", err)
		}
		if err := r.Create(ctx, job); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return ctrl.Result{}, fmt.Errorf("create Job %s: %w", jobKey, err)
			}
		} else {
			r.record(buildRun, corev1.EventTypeNormal, "JobCreated", "Created runner Job "+job.Name)
		}
		return ctrl.Result{}, r.markQueued(ctx, buildRun, job)
	}

	return ctrl.Result{}, r.syncStatusFromJob(ctx, buildRun, job)
}

func (r *BuildRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Scheme = mgr.GetScheme()
	r.Recorder = mgr.GetEventRecorderFor("buildrun-controller")
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

func (r *BuildRunReconciler) buildJob(buildRun *cicdv1alpha1.BuildRun, project *cicdv1alpha1.Project, repository *cicdv1alpha1.Repository, template *cicdv1alpha1.PipelineTemplate) *batchv1.Job {
	labels := map[string]string{
		"app.kubernetes.io/name":      "cloudivision",
		"app.kubernetes.io/component": "runner",
		"cloudivision.io/buildrun":    buildRun.Name,
		"cloudivision.io/project":     project.Name,
	}
	backoffLimit := defaultBackoffLimit
	ttlSeconds := defaultTTLSecondsAfterFinished
	var activeDeadlineSeconds *int64
	if template.Spec.Resources.TimeoutSeconds > 0 {
		timeout := int64(template.Spec.Resources.TimeoutSeconds)
		activeDeadlineSeconds = &timeout
	}
	runAsNonRoot := true
	allowPrivilegeEscalation := false
	privileged := false
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobNameForBuildRun(buildRun.Name),
			Namespace: buildRun.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: project.Spec.ServiceAccountName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
					},
					Containers: []corev1.Container{
						{
							Name:      "runner",
							Image:     runnerImage(),
							Env:       runnerEnv(buildRun, project, repository),
							Resources: resourceRequirements(template.Spec.Resources),
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             &runAsNonRoot,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								Privileged:               &privileged,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
				},
			},
		},
	}
	return job
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

func (r *BuildRunReconciler) markQueued(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, job *batchv1.Job) error {
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseQueued
	buildRun.Status.ObservedGeneration = buildRun.Generation
	buildRun.Status.JobRef = cicdv1alpha1.ObjectRef{Name: job.Name, Namespace: job.Namespace}
	domain.SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               "Queued",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: buildRun.Generation,
		Reason:             "JobCreated",
		Message:            "Runner Job has been created.",
	})
	return r.updateBuildRunStatus(ctx, buildRun)
}

func (r *BuildRunReconciler) syncStatusFromJob(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, job *batchv1.Job) error {
	buildRun.Status.JobRef = cicdv1alpha1.ObjectRef{Name: job.Name, Namespace: job.Namespace}
	now := metav1.Now()

	if job.Status.Succeeded > 0 {
		if err := domain.MarkBuildRunSucceeded(buildRun, now, buildRun.Spec.Image); err != nil {
			return err
		}
		r.record(buildRun, corev1.EventTypeNormal, "BuildSucceeded", "Runner Job completed successfully")
		return r.updateBuildRunStatus(ctx, buildRun)
	}

	if jobFailed(job) {
		if err := domain.MarkBuildRunFailed(buildRun, now, "JobFailed", "Runner Job failed"); err != nil {
			return err
		}
		r.record(buildRun, corev1.EventTypeWarning, "BuildFailed", "Runner Job failed")
		return r.updateBuildRunStatus(ctx, buildRun)
	}

	if job.Status.Active > 0 {
		if err := domain.MarkBuildRunStarted(buildRun, now); err != nil {
			return err
		}
		r.record(buildRun, corev1.EventTypeNormal, "BuildStarted", "Runner Job is running")
		return r.updateBuildRunStatus(ctx, buildRun)
	}

	if buildRun.Status.Phase == "" {
		buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseQueued
		buildRun.Status.ObservedGeneration = buildRun.Generation
		return r.updateBuildRunStatus(ctx, buildRun)
	}
	return nil
}

func (r *BuildRunReconciler) updateBuildRunStatus(ctx context.Context, buildRun *cicdv1alpha1.BuildRun) error {
	if err := r.Status().Update(ctx, buildRun); err != nil {
		return fmt.Errorf("update BuildRun status: %w", err)
	}
	return nil
}

func (r *BuildRunReconciler) record(buildRun *cicdv1alpha1.BuildRun, eventType, reason, message string) {
	if r.Recorder != nil {
		r.Recorder.Event(buildRun, eventType, reason, message)
	}
}

func runnerImage() string {
	if image := os.Getenv("CLOU_DIVISION_RUNNER_IMAGE"); image != "" {
		return image
	}
	return defaultRunnerImage
}

func runnerEnv(buildRun *cicdv1alpha1.BuildRun, project *cicdv1alpha1.Project, repository *cicdv1alpha1.Repository) []corev1.EnvVar {
	return []corev1.EnvVar{
		{Name: "BUILD_RUN_NAME", Value: buildRun.Name},
		{Name: "BUILD_RUN_NAMESPACE", Value: buildRun.Namespace},
		{Name: "PROJECT_NAME", Value: project.Name},
		{Name: "REPOSITORY_URL", Value: repository.Spec.URL},
		{Name: "REVISION", Value: buildRun.Spec.Revision},
		{Name: "BRANCH", Value: buildRun.Spec.Branch},
		{Name: "PIPELINE_TEMPLATE_NAME", Value: buildRun.Spec.PipelineTemplateRef},
		{Name: "IMAGE_REPOSITORY", Value: buildRun.Spec.Image.Repository},
		{Name: "IMAGE_TAG", Value: buildRun.Spec.Image.Tag},
		{Name: "GITOPS_ENABLED", Value: strconv.FormatBool(buildRun.Spec.GitOps.Enabled)},
	}
}

func resourceRequirements(resources cicdv1alpha1.PipelineResourceSpec) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	addQuantity(requirements.Requests, corev1.ResourceCPU, resources.CPURequest)
	addQuantity(requirements.Limits, corev1.ResourceCPU, resources.CPULimit)
	addQuantity(requirements.Requests, corev1.ResourceMemory, resources.MemoryRequest)
	addQuantity(requirements.Limits, corev1.ResourceMemory, resources.MemoryLimit)
	if len(requirements.Requests) == 0 {
		requirements.Requests = nil
	}
	if len(requirements.Limits) == 0 {
		requirements.Limits = nil
	}
	return requirements
}

func addQuantity(list corev1.ResourceList, name corev1.ResourceName, value string) {
	if value == "" {
		return
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return
	}
	list[name] = quantity
}

func jobNameForBuildRun(buildRunName string) string {
	name := strings.ToLower(buildRunName) + "-runner"
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

func isTerminalBuildRunPhase(phase cicdv1alpha1.BuildRunPhase) bool {
	return phase == cicdv1alpha1.BuildRunPhaseSucceeded ||
		phase == cicdv1alpha1.BuildRunPhaseFailed ||
		phase == cicdv1alpha1.BuildRunPhaseCancelled
}

func jobFailed(job *batchv1.Job) bool {
	if job.Status.Failed == 0 {
		return false
	}
	if job.Spec.BackoffLimit == nil {
		return job.Status.Failed > defaultBackoffLimit
	}
	return job.Status.Failed > *job.Spec.BackoffLimit
}

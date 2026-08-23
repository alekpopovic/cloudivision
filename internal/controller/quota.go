package controller

import (
	"context"
	"fmt"
	"sort"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/observability"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type quotaDecision struct {
	queue  bool
	deny   bool
	active int
	queued int
}

func (r *BuildRunReconciler) evaluateProjectQuota(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, project *cicdv1alpha1.Project) (quotaDecision, error) {
	if project.Spec.Quotas == nil || project.Spec.Quotas.MaxConcurrentBuildRuns <= 0 || buildRun.Status.JobRef.Name != "" || buildRun.Status.PipelineRunRef.Name != "" {
		return quotaDecision{}, nil
	}
	var runs cicdv1alpha1.BuildRunList
	if err := r.List(ctx, &runs, client.InNamespace(buildRun.Namespace)); err != nil {
		return quotaDecision{}, fmt.Errorf("list BuildRuns for project quota: %w", err)
	}
	decision := quotaDecision{}
	waiting := []*cicdv1alpha1.BuildRun{}
	for i := range runs.Items {
		candidate := &runs.Items[i]
		if candidate.Spec.ProjectRef != buildRun.Spec.ProjectRef || isTerminalBuildRunPhase(candidate.Status.Phase) {
			continue
		}
		if candidate.Status.Phase == cicdv1alpha1.BuildRunPhaseRunning || candidate.Status.Phase == cicdv1alpha1.BuildRunPhaseQueued && (candidate.Status.JobRef.Name != "" || candidate.Status.PipelineRunRef.Name != "") {
			decision.active++
			continue
		}
		if candidate.Status.JobRef.Name == "" && candidate.Status.PipelineRunRef.Name == "" {
			waiting = append(waiting, candidate)
		}
	}
	sort.Slice(waiting, func(i, j int) bool {
		if waiting[i].CreationTimestamp.Time.Equal(waiting[j].CreationTimestamp.Time) {
			return waiting[i].Name < waiting[j].Name
		}
		return waiting[i].CreationTimestamp.Time.Before(waiting[j].CreationTimestamp.Time)
	})
	available := project.Spec.Quotas.MaxConcurrentBuildRuns - decision.active
	if available < 0 {
		available = 0
	}
	targetRank := -1
	for i, candidate := range waiting {
		if candidate.Name == buildRun.Name {
			targetRank = i
			break
		}
	}
	admitted := available
	if admitted > len(waiting) {
		admitted = len(waiting)
	}
	decision.queued = len(waiting) - admitted
	observability.ActiveBuildRuns.WithLabelValues(project.Name).Set(float64(decision.active))
	observability.QueuedBuildRuns.WithLabelValues(project.Name).Set(float64(decision.queued))
	if targetRank >= 0 && targetRank < available {
		return decision, nil
	}
	queueRank := targetRank - available
	if project.Spec.Quotas.MaxQueuedBuildRuns > 0 && queueRank >= project.Spec.Quotas.MaxQueuedBuildRuns {
		decision.deny = true
		return decision, nil
	}
	decision.queue = true
	return decision, nil
}

func (r *BuildRunReconciler) markQuotaQueued(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, active, maximum int) error {
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseQueued
	buildRun.Status.ObservedGeneration = buildRun.Generation
	buildRun.Status.Failure = cicdv1alpha1.FailureStatus{}
	domain.SetCondition(&buildRun.Status.Conditions, metav1.Condition{Type: "Queued", Status: metav1.ConditionTrue, ObservedGeneration: buildRun.Generation, Reason: "ConcurrencyQuotaReached", Message: fmt.Sprintf("Project has %d active BuildRuns; waiting for one of %d slots.", active, maximum), LastTransitionTime: metav1.Now()})
	r.record(buildRun, corev1.EventTypeNormal, "BuildQueuedByQuota", "BuildRun is waiting for a project concurrency slot.")
	return r.updateBuildRunStatus(ctx, buildRun)
}

func (r *BuildRunReconciler) markQuotaExceeded(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, queued, maximum int) error {
	message := fmt.Sprintf("Project queue contains %d BuildRuns and the configured maximum is %d.", queued, maximum)
	now := metav1.Now()
	if err := domain.MarkBuildRunFailed(buildRun, now, "QuotaExceeded", message); err != nil {
		return err
	}
	observability.QuotaDeniedBuildRuns.Inc()
	r.record(buildRun, corev1.EventTypeWarning, "QuotaExceeded", message)
	return r.updateBuildRunStatus(ctx, buildRun)
}

func applyProjectWorkloadQuotas(template *cicdv1alpha1.PipelineTemplate, quotas *cicdv1alpha1.ProjectQuotaSpec) *cicdv1alpha1.PipelineTemplate {
	copy := template.DeepCopy()
	if quotas == nil {
		return copy
	}
	if quotas.MaxBuildDurationSeconds > 0 && (copy.Spec.Resources.TimeoutSeconds == 0 || quotas.MaxBuildDurationSeconds < copy.Spec.Resources.TimeoutSeconds) {
		copy.Spec.Resources.TimeoutSeconds = quotas.MaxBuildDurationSeconds
	}
	copy.Spec.Resources.CPULimit = stricterQuantity(copy.Spec.Resources.CPULimit, quotas.MaxCPU)
	copy.Spec.Resources.MemoryLimit = stricterQuantity(copy.Spec.Resources.MemoryLimit, quotas.MaxMemory)
	copy.Spec.Resources.CPURequest = noGreaterThan(copy.Spec.Resources.CPURequest, copy.Spec.Resources.CPULimit)
	copy.Spec.Resources.MemoryRequest = noGreaterThan(copy.Spec.Resources.MemoryRequest, copy.Spec.Resources.MemoryLimit)
	return copy
}

func stricterQuantity(current, maximum string) string {
	if maximum == "" {
		return current
	}
	max, err := resource.ParseQuantity(maximum)
	if err != nil {
		return current
	}
	value, err := resource.ParseQuantity(current)
	if current == "" || err != nil || value.Cmp(max) > 0 {
		return maximum
	}
	return current
}

func noGreaterThan(request, limit string) string {
	if request == "" || limit == "" {
		return request
	}
	req, reqErr := resource.ParseQuantity(request)
	lim, limErr := resource.ParseQuantity(limit)
	if reqErr == nil && limErr == nil && req.Cmp(lim) > 0 {
		return limit
	}
	return request
}

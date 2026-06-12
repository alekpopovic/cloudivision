package domain

import (
	"fmt"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConditionReady     = "Ready"
	ConditionRunning   = "Running"
	ConditionSucceeded = "Succeeded"
	ConditionFailed    = "Failed"
)

// SetCondition inserts or updates a condition while preserving Kubernetes condition semantics.
func SetCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	if condition.LastTransitionTime.IsZero() {
		condition.LastTransitionTime = metav1.Now()
	}
	metaSetStatusCondition(conditions, condition)
}

// PhaseFromConditions computes a simple Pending/Ready/Error style phase from status conditions.
func PhaseFromConditions(conditions []metav1.Condition, pendingPhase, readyPhase, errorPhase string) string {
	if findCondition(conditions, ConditionFailed, metav1.ConditionTrue) != nil {
		return errorPhase
	}
	if ready := findCondition(conditions, ConditionReady, metav1.ConditionTrue); ready != nil {
		return readyPhase
	}
	return pendingPhase
}

func MarkBuildRunStarted(buildRun *cicdv1alpha1.BuildRun, now metav1.Time) error {
	if err := ValidateBuildRunPhaseTransition(buildRun.Status.Phase, cicdv1alpha1.BuildRunPhaseRunning); err != nil {
		return err
	}
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseRunning
	buildRun.Status.ObservedGeneration = buildRun.Generation
	if buildRun.Status.StartedAt == nil {
		buildRun.Status.StartedAt = &now
	}
	SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               ConditionRunning,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: buildRun.Generation,
		Reason:             "BuildRunStarted",
		Message:            "BuildRun execution has started.",
		LastTransitionTime: now,
	})
	return nil
}

func MarkBuildRunSucceeded(buildRun *cicdv1alpha1.BuildRun, now metav1.Time, image cicdv1alpha1.ImageRef) error {
	if err := ValidateBuildRunPhaseTransition(buildRun.Status.Phase, cicdv1alpha1.BuildRunPhaseSucceeded); err != nil {
		return err
	}
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseSucceeded
	buildRun.Status.ObservedGeneration = buildRun.Generation
	buildRun.Status.CompletedAt = &now
	buildRun.Status.Image = image
	buildRun.Status.Failure = cicdv1alpha1.FailureStatus{}
	SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               ConditionSucceeded,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: buildRun.Generation,
		Reason:             "BuildRunSucceeded",
		Message:            "BuildRun completed successfully.",
		LastTransitionTime: now,
	})
	SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               ConditionRunning,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: buildRun.Generation,
		Reason:             "BuildRunCompleted",
		Message:            "BuildRun is no longer running.",
		LastTransitionTime: now,
	})
	return nil
}

func MarkBuildRunFailed(buildRun *cicdv1alpha1.BuildRun, now metav1.Time, reason, message string) error {
	if err := ValidateBuildRunPhaseTransition(buildRun.Status.Phase, cicdv1alpha1.BuildRunPhaseFailed); err != nil {
		return err
	}
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseFailed
	buildRun.Status.ObservedGeneration = buildRun.Generation
	buildRun.Status.CompletedAt = &now
	buildRun.Status.Failure = cicdv1alpha1.FailureStatus{
		Reason:  reason,
		Message: message,
	}
	SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: buildRun.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
	SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               ConditionRunning,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: buildRun.Generation,
		Reason:             "BuildRunCompleted",
		Message:            "BuildRun is no longer running.",
		LastTransitionTime: now,
	})
	return nil
}

func ValidateBuildRunPhaseTransition(from, to cicdv1alpha1.BuildRunPhase) error {
	if from == to {
		return nil
	}
	allowed := map[cicdv1alpha1.BuildRunPhase][]cicdv1alpha1.BuildRunPhase{
		"": {
			cicdv1alpha1.BuildRunPhasePending,
			cicdv1alpha1.BuildRunPhaseQueued,
			cicdv1alpha1.BuildRunPhaseRunning,
			cicdv1alpha1.BuildRunPhaseSucceeded,
			cicdv1alpha1.BuildRunPhaseFailed,
		},
		cicdv1alpha1.BuildRunPhasePending: {
			cicdv1alpha1.BuildRunPhaseQueued,
			cicdv1alpha1.BuildRunPhaseRunning,
			cicdv1alpha1.BuildRunPhaseSucceeded,
			cicdv1alpha1.BuildRunPhaseCancelled,
			cicdv1alpha1.BuildRunPhaseFailed,
		},
		cicdv1alpha1.BuildRunPhaseQueued: {
			cicdv1alpha1.BuildRunPhaseRunning,
			cicdv1alpha1.BuildRunPhaseSucceeded,
			cicdv1alpha1.BuildRunPhaseCancelled,
			cicdv1alpha1.BuildRunPhaseFailed,
		},
		cicdv1alpha1.BuildRunPhaseRunning: {
			cicdv1alpha1.BuildRunPhaseSucceeded,
			cicdv1alpha1.BuildRunPhaseFailed,
			cicdv1alpha1.BuildRunPhaseCancelled,
		},
	}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return nil
		}
	}
	return fmt.Errorf("invalid BuildRun phase transition from %q to %q", from, to)
}

func findCondition(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType && conditions[i].Status == status {
			return &conditions[i]
		}
	}
	return nil
}

func metaSetStatusCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	for i := range *conditions {
		if (*conditions)[i].Type == condition.Type {
			if (*conditions)[i].Status == condition.Status && !(*conditions)[i].LastTransitionTime.IsZero() {
				condition.LastTransitionTime = (*conditions)[i].LastTransitionTime
			}
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}

package domain

import (
	"testing"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetConditionUpdatesExistingCondition(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, 6, 12, 1, 0, 0, 0, time.UTC))
	conditions := []metav1.Condition{
		{
			Type:               ConditionReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: 1,
			Reason:             "Pending",
			Message:            "Not ready yet.",
			LastTransitionTime: now,
		},
	}

	SetCondition(&conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 2,
		Reason:             "Ready",
		Message:            "Ready.",
	})

	if len(conditions) != 1 {
		t.Fatalf("len(conditions) = %d, want 1", len(conditions))
	}
	if conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("condition status = %s, want %s", conditions[0].Status, metav1.ConditionTrue)
	}
	if conditions[0].Reason != "Ready" {
		t.Fatalf("condition reason = %q, want Ready", conditions[0].Reason)
	}
}

func TestPhaseFromConditions(t *testing.T) {
	tests := []struct {
		name       string
		conditions []metav1.Condition
		want       string
	}{
		{
			name: "pending without conditions",
			want: "Pending",
		},
		{
			name: "ready wins without failure",
			conditions: []metav1.Condition{
				{Type: ConditionReady, Status: metav1.ConditionTrue},
			},
			want: "Ready",
		},
		{
			name: "failure wins over ready",
			conditions: []metav1.Condition{
				{Type: ConditionReady, Status: metav1.ConditionTrue},
				{Type: ConditionFailed, Status: metav1.ConditionTrue},
			},
			want: "Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PhaseFromConditions(tt.conditions, "Pending", "Ready", "Error")
			if got != tt.want {
				t.Fatalf("PhaseFromConditions() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkBuildRunLifecycle(t *testing.T) {
	started := metav1.NewTime(time.Date(2026, 6, 12, 1, 0, 0, 0, time.UTC))
	completed := metav1.NewTime(started.Add(5 * time.Minute))
	buildRun := &cicdv1alpha1.BuildRun{}
	buildRun.Generation = 7
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhasePending

	if err := MarkBuildRunStarted(buildRun, started); err != nil {
		t.Fatalf("MarkBuildRunStarted() error = %v", err)
	}
	if buildRun.Status.Phase != cicdv1alpha1.BuildRunPhaseRunning {
		t.Fatalf("phase after start = %q, want Running", buildRun.Status.Phase)
	}
	if buildRun.Status.StartedAt == nil || !buildRun.Status.StartedAt.Equal(&started) {
		t.Fatalf("startedAt = %v, want %v", buildRun.Status.StartedAt, started)
	}

	image := cicdv1alpha1.ImageRef{
		Repository: "ghcr.io/cloudivision/app",
		Tag:        "main",
		Digest:     "sha256:example",
	}
	if err := MarkBuildRunSucceeded(buildRun, completed, image); err != nil {
		t.Fatalf("MarkBuildRunSucceeded() error = %v", err)
	}
	if buildRun.Status.Phase != cicdv1alpha1.BuildRunPhaseSucceeded {
		t.Fatalf("phase after success = %q, want Succeeded", buildRun.Status.Phase)
	}
	if buildRun.Status.Image != image {
		t.Fatalf("image = %#v, want %#v", buildRun.Status.Image, image)
	}
	if buildRun.Status.CompletedAt == nil || !buildRun.Status.CompletedAt.Equal(&completed) {
		t.Fatalf("completedAt = %v, want %v", buildRun.Status.CompletedAt, completed)
	}
}

func TestMarkBuildRunFailed(t *testing.T) {
	now := metav1.NewTime(time.Date(2026, 6, 12, 1, 0, 0, 0, time.UTC))
	buildRun := &cicdv1alpha1.BuildRun{}
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseRunning

	if err := MarkBuildRunFailed(buildRun, now, "JobFailed", "The build job failed."); err != nil {
		t.Fatalf("MarkBuildRunFailed() error = %v", err)
	}
	if buildRun.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed {
		t.Fatalf("phase after failure = %q, want Failed", buildRun.Status.Phase)
	}
	if buildRun.Status.Failure.Reason != "JobFailed" {
		t.Fatalf("failure reason = %q, want JobFailed", buildRun.Status.Failure.Reason)
	}
}

func TestValidateBuildRunPhaseTransitionRejectsTerminalTransition(t *testing.T) {
	err := ValidateBuildRunPhaseTransition(cicdv1alpha1.BuildRunPhaseSucceeded, cicdv1alpha1.BuildRunPhaseRunning)
	if err == nil {
		t.Fatal("ValidateBuildRunPhaseTransition() error = nil, want error")
	}
}

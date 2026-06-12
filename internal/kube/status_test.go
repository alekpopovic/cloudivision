package kube

import (
	"context"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpdateStatusWithRetryUpdatesLatestObject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	ctx := context.Background()
	buildRun := &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "build-1", Namespace: "ci"},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}).
		WithObjects(buildRun).
		Build()

	desired := &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "build-1", Namespace: "ci"},
		Status: cicdv1alpha1.BuildRunStatus{
			Phase: cicdv1alpha1.BuildRunPhaseRunning,
			Conditions: []metav1.Condition{{
				Type:               "Running",
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 1,
				Reason:             "Started",
				Message:            "Build is running.",
				LastTransitionTime: metav1.Now(),
			}},
		},
	}

	if err := UpdateStatusWithRetry(ctx, c, desired); err != nil {
		t.Fatalf("UpdateStatusWithRetry() error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: "ci", Name: "build-1"}, updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseRunning {
		t.Fatalf("phase = %q, want Running", updated.Status.Phase)
	}
}

package kube

import (
	"context"
	"errors"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		Spec:       cicdv1alpha1.BuildRunSpec{Revision: "stored-revision"},
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}).
		WithObjects(buildRun).
		Build()

	desired := &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "build-1", Namespace: "ci"},
		Spec:       cicdv1alpha1.BuildRunSpec{Revision: "stale-desired-revision"},
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
	if updated.Spec.Revision != "stored-revision" {
		t.Fatalf("spec.revision = %q, status update overwrote spec", updated.Spec.Revision)
	}
}

func TestUpdateStatusWithRetryPreservesConcurrentConditions(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}
	ctx := context.Background()
	stored := &cicdv1alpha1.BuildRun{ObjectMeta: metav1.ObjectMeta{Name: "build-conflict", Namespace: "ci"}}
	base := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}).
		WithObjects(stored).
		Build()
	conflicting := &conditionConflictClient{Client: base}
	desired := stored.DeepCopy()
	desired.Status.Phase = cicdv1alpha1.BuildRunPhaseRunning
	desired.Status.Conditions = []metav1.Condition{{
		Type:               "Running",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 1,
		Reason:             "Started",
		Message:            "Build started.",
		LastTransitionTime: metav1.Now(),
	}}

	if err := UpdateStatusWithRetry(ctx, conflicting, desired); err != nil {
		t.Fatalf("UpdateStatusWithRetry() error = %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := base.Get(ctx, types.NamespacedName{Namespace: "ci", Name: "build-conflict"}, updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	for _, conditionType := range []string{"ExternalAudit", "Running"} {
		if !containsCondition(updated.Status.Conditions, conditionType) {
			t.Fatalf("conditions = %#v, want %s preserved", updated.Status.Conditions, conditionType)
		}
	}
}

func containsCondition(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

type conditionConflictClient struct {
	client.Client
	injected bool
}

func (c *conditionConflictClient) Status() client.SubResourceWriter {
	return conditionConflictWriter{delegate: c.Client.Status(), parent: c}
}

type conditionConflictWriter struct {
	delegate client.SubResourceWriter
	parent   *conditionConflictClient
}

func (w conditionConflictWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	return w.delegate.Create(ctx, obj, subResource, opts...)
}

func (w conditionConflictWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if !w.parent.injected {
		w.parent.injected = true
		external := &cicdv1alpha1.BuildRun{}
		if err := w.parent.Get(ctx, client.ObjectKeyFromObject(obj), external); err != nil {
			return err
		}
		external.Status.Conditions = []metav1.Condition{{
			Type:               "ExternalAudit",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
			Reason:             "Recorded",
			Message:            "Concurrent actor condition.",
			LastTransitionTime: metav1.Now(),
		}}
		if err := w.delegate.Update(ctx, external); err != nil {
			return err
		}
		return apierrors.NewConflict(schema.GroupResource{Group: "cicd.cloudivision.io", Resource: "buildruns"}, obj.GetName(), errors.New("injected conflict"))
	}
	return w.delegate.Update(ctx, obj, opts...)
}

func (w conditionConflictWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return w.delegate.Patch(ctx, obj, patch, opts...)
}

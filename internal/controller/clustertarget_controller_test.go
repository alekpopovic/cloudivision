package controller

import (
	"context"
	"errors"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	clusterprovider "github.com/cloudivision/cloudivision/internal/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type staticClusterChecker struct {
	result clusterprovider.Result
	err    error
}

func (s staticClusterChecker) Check(context.Context, *cicdv1alpha1.ClusterTarget) (clusterprovider.Result, error) {
	return s.result, s.err
}

func TestClusterTargetHealthStatus(t *testing.T) {
	for _, test := range []struct {
		name      string
		checker   staticClusterChecker
		phase     cicdv1alpha1.ClusterTargetPhase
		reachable bool
	}{{"ready", staticClusterChecker{result: clusterprovider.Result{Version: "v1.34.2"}}, cicdv1alpha1.ClusterTargetPhaseReady, true}, {"unreachable", staticClusterChecker{err: errors.New("connection refused")}, cicdv1alpha1.ClusterTargetPhaseUnreachable, false}} {
		t.Run(test.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			target := &cicdv1alpha1.ClusterTarget{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "ci", Generation: 2}}
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(target).WithObjects(target).Build()
			r := ClusterTargetReconciler{Client: fakeClient, Checker: test.checker}
			if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "target", Namespace: "ci"}}); err != nil {
				t.Fatal(err)
			}
			got := &cicdv1alpha1.ClusterTarget{}
			if err := fakeClient.Get(context.Background(), types.NamespacedName{Name: "target", Namespace: "ci"}, got); err != nil {
				t.Fatal(err)
			}
			if got.Status.Phase != test.phase || got.Status.Reachable != test.reachable || got.Status.ObservedGeneration != 2 {
				t.Fatalf("status = %#v", got.Status)
			}
		})
	}
}

package controller

import (
	"context"
	"fmt"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	clusterprovider "github.com/cloudivision/cloudivision/internal/cluster"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerconfig "sigs.k8s.io/controller-runtime/pkg/controller"
)

const clusterTargetRequeue = 2 * time.Minute

type ClusterTargetReconciler struct {
	client.Client
	Checker                 clusterprovider.Checker
	MaxConcurrentReconciles int
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=clustertargets,verbs=get;list;watch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=clustertargets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
func (r *ClusterTargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	target := &cicdv1alpha1.ClusterTarget{}
	if err := r.Get(ctx, req.NamespacedName, target); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get ClusterTarget %s: %w", req.NamespacedName, err)
	}
	result, checkErr := r.Checker.Check(ctx, target)
	target.Status.ObservedGeneration = target.Generation
	target.Status.Version = result.Version
	target.Status.Reachable = checkErr == nil
	condition := metav1.Condition{Type: "Reachable", ObservedGeneration: target.Generation, LastTransitionTime: metav1.Now()}
	if checkErr != nil {
		target.Status.Phase = cicdv1alpha1.ClusterTargetPhaseUnreachable
		condition.Status = metav1.ConditionFalse
		condition.Reason = "HealthCheckFailed"
		condition.Message = checkErr.Error()
	} else {
		target.Status.Phase = cicdv1alpha1.ClusterTargetPhaseReady
		condition.Status = metav1.ConditionTrue
		condition.Reason = "HealthCheckSucceeded"
		condition.Message = "Kubernetes API is reachable"
	}
	apiMeta.SetStatusCondition(&target.Status.Conditions, condition)
	if err := r.Status().Update(ctx, target); err != nil {
		return ctrl.Result{}, fmt.Errorf("update ClusterTarget %s status: %w", req.NamespacedName, err)
	}
	return ctrl.Result{RequeueAfter: clusterTargetRequeue}, nil
}

func (r *ClusterTargetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&cicdv1alpha1.ClusterTarget{}).WithOptions(controllerconfig.Options{MaxConcurrentReconciles: normalizedConcurrency(r.MaxConcurrentReconciles)}).Complete(r)
}

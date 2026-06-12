package controller

import (
	"context"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BuildRunReconciler reconciles BuildRun resources.
type BuildRunReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *BuildRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.FromContext(ctx).V(1).Info("reconciling BuildRun", "name", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *BuildRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cicdv1alpha1.BuildRun{}).
		Complete(r)
}

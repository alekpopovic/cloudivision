package controller

import (
	"context"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ProjectReconciler reconciles Project resources.
type ProjectReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.FromContext(ctx).V(1).Info("reconciling Project", "name", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cicdv1alpha1.Project{}).
		Complete(r)
}

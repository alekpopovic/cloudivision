package controller

import (
	"context"
	"fmt"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/kube"
	"github.com/cloudivision/cloudivision/internal/observability"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultRunnerServiceAccount = "cloudivision-runner"
	runnerRoleName              = "cloudivision-runner"
	defaultDenyNetworkPolicy    = "cloudivision-default-deny"
	egressAllowListPolicy       = "cloudivision-egress-allow-list"
)

// ProjectReconciler reconciles Project resources.
type ProjectReconciler struct {
	client.Client
}

// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=buildruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cicd.cloudivision.io,resources=repositories;pipelinetemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	started := time.Now()
	defer func() {
		observability.ObserveReconcile("project", started, err)
	}()
	return r.reconcile(ctx, req)
}

func (r *ProjectReconciler) reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("controller", "project", "namespace", req.Namespace, "project", req.Name)
	project := &cicdv1alpha1.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get Project %s: %w", req.NamespacedName, err)
	}
	logger = logger.WithValues("correlationId", project.Annotations[observability.CorrelationIDAnno])

	if project.Spec.Isolation.CreateNamespace {
		logger.Info("ensuring project namespace", "targetNamespace", project.Spec.Namespace)
		if err := r.ensureNamespace(ctx, project); err != nil {
			return ctrl.Result{}, r.markProjectError(ctx, project, err)
		}
	}
	if err := r.ensureRunnerRBAC(ctx, project); err != nil {
		return ctrl.Result{}, r.markProjectError(ctx, project, err)
	}
	if err := r.ensureNetworkPolicy(ctx, project); err != nil {
		return ctrl.Result{}, r.markProjectError(ctx, project, err)
	}
	return ctrl.Result{}, r.markProjectReady(ctx, project)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cicdv1alpha1.Project{}).
		Complete(r)
}

func (r *ProjectReconciler) ensureNamespace(ctx context.Context, project *cicdv1alpha1.Project) error {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: project.Spec.Namespace}}
	key := client.ObjectKey{Name: project.Spec.Namespace}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, namespace, func() error {
		labels := namespace.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels["app.kubernetes.io/managed-by"] = "cloudivision"
		labels["cloudivision.io/project"] = project.Name
		if project.Spec.Isolation.PodSecurityLevel != "" {
			level := string(project.Spec.Isolation.PodSecurityLevel)
			labels["pod-security.kubernetes.io/enforce"] = level
			labels["pod-security.kubernetes.io/audit"] = level
			labels["pod-security.kubernetes.io/warn"] = level
		}
		namespace.SetLabels(labels)
		return nil
	})
	if err != nil {
		return fmt.Errorf("ensure Namespace %s: %w", key.Name, err)
	}
	return nil
}

func (r *ProjectReconciler) ensureRunnerRBAC(ctx context.Context, project *cicdv1alpha1.Project) error {
	namespace := project.Spec.Namespace
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: runnerServiceAccountName(project), Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, serviceAccount, func() error {
		labels := managedLabels(project)
		serviceAccount.Labels = mergeLabels(serviceAccount.Labels, labels)
		return nil
	}); err != nil {
		return fmt.Errorf("ensure runner ServiceAccount %s/%s: %w", namespace, serviceAccount.Name, err)
	}

	role := &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: runnerRoleName, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		role.Labels = mergeLabels(role.Labels, managedLabels(project))
		role.Rules = runnerPolicyRules()
		return nil
	}); err != nil {
		return fmt.Errorf("ensure runner Role %s/%s: %w", namespace, runnerRoleName, err)
	}

	binding := &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: runnerRoleName, Namespace: namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, binding, func() error {
		binding.Labels = mergeLabels(binding.Labels, managedLabels(project))
		binding.Subjects = []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      serviceAccount.Name,
			Namespace: namespace,
		}}
		binding.RoleRef = rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     runnerRoleName,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("ensure runner RoleBinding %s/%s: %w", namespace, runnerRoleName, err)
	}
	return nil
}

func (r *ProjectReconciler) ensureNetworkPolicy(ctx context.Context, project *cicdv1alpha1.Project) error {
	switch project.Spec.Isolation.NetworkPolicyMode {
	case cicdv1alpha1.NetworkPolicyModeDefaultDeny:
		policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: defaultDenyNetworkPolicy, Namespace: project.Spec.Namespace}}
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
			policy.Labels = mergeLabels(policy.Labels, managedLabels(project))
			policy.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
					networkingv1.PolicyTypeEgress,
				},
			}
			return nil
		}); err != nil {
			return fmt.Errorf("ensure default deny NetworkPolicy %s/%s: %w", project.Spec.Namespace, defaultDenyNetworkPolicy, err)
		}
	case cicdv1alpha1.NetworkPolicyModeEgressAllowList:
		policy := &networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: egressAllowListPolicy, Namespace: project.Spec.Namespace}}
		if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, policy, func() error {
			policy.Labels = mergeLabels(policy.Labels, managedLabels(project))
			annotations := policy.GetAnnotations()
			if annotations == nil {
				annotations = map[string]string{}
			}
			annotations["cloudivision.io/egress-allow-list-extension"] = "Project.spec currently has no CIDR/DNS allow-list fields; extend ProjectIsolation before opening egress."
			policy.SetAnnotations(annotations)
			policy.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
			}
			return nil
		}); err != nil {
			return fmt.Errorf("ensure egress allow-list NetworkPolicy skeleton %s/%s: %w", project.Spec.Namespace, egressAllowListPolicy, err)
		}
	}
	return nil
}

func (r *ProjectReconciler) markProjectReady(ctx context.Context, project *cicdv1alpha1.Project) error {
	project.Status.Phase = cicdv1alpha1.ProjectPhaseReady
	project.Status.NamespaceReady = project.Spec.Isolation.CreateNamespace
	project.Status.ObservedGeneration = project.Generation
	domain.SetCondition(&project.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: project.Generation,
		Reason:             "ProjectReady",
		Message:            "Project isolation resources are ready.",
	})
	if err := kube.UpdateStatusWithRetry(ctx, r.Client, project); err != nil {
		return fmt.Errorf("update Project status: %w", err)
	}
	return nil
}

func (r *ProjectReconciler) markProjectError(ctx context.Context, project *cicdv1alpha1.Project, err error) error {
	project.Status.Phase = cicdv1alpha1.ProjectPhaseError
	project.Status.ObservedGeneration = project.Generation
	domain.SetCondition(&project.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: project.Generation,
		Reason:             "ReconcileFailed",
		Message:            err.Error(),
	})
	if updateErr := kube.UpdateStatusWithRetry(ctx, r.Client, project); updateErr != nil {
		return fmt.Errorf("update Project error status after %v: %w", err, updateErr)
	}
	return err
}

func runnerServiceAccountName(project *cicdv1alpha1.Project) string {
	if project.Spec.ServiceAccountName != "" {
		return project.Spec.ServiceAccountName
	}
	return defaultRunnerServiceAccount
}

func runnerPolicyRules() []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"cicd.cloudivision.io"},
			Resources: []string{"buildruns"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{"cicd.cloudivision.io"},
			Resources: []string{"buildruns/status"},
			Verbs:     []string{"get", "update", "patch"},
		},
		{
			APIGroups: []string{"cicd.cloudivision.io"},
			Resources: []string{"repositories", "pipelinetemplates"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create", "update", "patch"},
		},
	}
}

func managedLabels(project *cicdv1alpha1.Project) map[string]string {
	return map[string]string{
		"app.kubernetes.io/managed-by": "cloudivision",
		"cloudivision.io/project":      project.Name,
	}
}

func mergeLabels(existing, desired map[string]string) map[string]string {
	merged := map[string]string{}
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range desired {
		merged[key] = value
	}
	return merged
}

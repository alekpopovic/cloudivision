package controller

import (
	"context"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProjectReconcileCreatesIsolationResources(t *testing.T) {
	ctx := context.Background()
	reconciler, project := newProjectReconciler(t, testIsolatedProject())

	if _, err := reconciler.Reconcile(ctx, projectRequestFor(project)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	namespace := &corev1.Namespace{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: project.Spec.Namespace}, namespace); err != nil {
		t.Fatalf("get Namespace error = %v", err)
	}
	if namespace.Labels["app.kubernetes.io/managed-by"] != "cloudivision" {
		t.Fatalf("managed-by label = %q", namespace.Labels["app.kubernetes.io/managed-by"])
	}
	if namespace.Labels["pod-security.kubernetes.io/enforce"] != "restricted" {
		t.Fatalf("pod security enforce label = %q", namespace.Labels["pod-security.kubernetes.io/enforce"])
	}

	serviceAccount := &corev1.ServiceAccount{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: "sample-runner", Namespace: project.Spec.Namespace}, serviceAccount); err != nil {
		t.Fatalf("get ServiceAccount error = %v", err)
	}

	role := &rbacv1.Role{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: runnerRoleName, Namespace: project.Spec.Namespace}, role); err != nil {
		t.Fatalf("get Role error = %v", err)
	}
	assertNoClusterAdminRules(t, role.Rules)
	if !roleAllows(role.Rules, "cicd.cloudivision.io", "buildruns/status", "update") {
		t.Fatalf("role rules = %#v, want buildruns/status update", role.Rules)
	}

	binding := &rbacv1.RoleBinding{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: runnerRoleName, Namespace: project.Spec.Namespace}, binding); err != nil {
		t.Fatalf("get RoleBinding error = %v", err)
	}
	if len(binding.Subjects) != 1 || binding.Subjects[0].Name != "sample-runner" {
		t.Fatalf("subjects = %#v, want sample-runner ServiceAccount", binding.Subjects)
	}

	policy := &networkingv1.NetworkPolicy{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: defaultDenyNetworkPolicy, Namespace: project.Spec.Namespace}, policy); err != nil {
		t.Fatalf("get NetworkPolicy error = %v", err)
	}
	if len(policy.Spec.Ingress) != 0 || len(policy.Spec.Egress) != 0 {
		t.Fatalf("default deny policy should have empty ingress and egress rules")
	}
}

func TestProjectReconcileCreatesEgressAllowListSkeleton(t *testing.T) {
	ctx := context.Background()
	project := testIsolatedProject()
	project.Spec.Isolation.NetworkPolicyMode = cicdv1alpha1.NetworkPolicyModeEgressAllowList
	reconciler, project := newProjectReconciler(t, project)

	if _, err := reconciler.Reconcile(ctx, projectRequestFor(project)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	policy := &networkingv1.NetworkPolicy{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: egressAllowListPolicy, Namespace: project.Spec.Namespace}, policy); err != nil {
		t.Fatalf("get NetworkPolicy error = %v", err)
	}
	if policy.Annotations["cloudivision.io/egress-allow-list-extension"] == "" {
		t.Fatalf("annotations = %#v, want extension note", policy.Annotations)
	}
	if len(policy.Spec.Egress) != 0 {
		t.Fatalf("egress rules = %#v, want empty skeleton until allow-list fields exist", policy.Spec.Egress)
	}
}

func newProjectReconciler(t *testing.T, project *cicdv1alpha1.Project) (*ProjectReconciler, *cicdv1alpha1.Project) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.Project{}).
		WithObjects(project).
		Build()
	return &ProjectReconciler{Client: fakeClient}, project
}

func testIsolatedProject() *cicdv1alpha1.Project {
	return &cicdv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-project", Namespace: "control"},
		Spec: cicdv1alpha1.ProjectSpec{
			DisplayName:        "Sample",
			OwnerTeam:          "platform",
			Namespace:          "sample-project",
			DefaultRegistry:    "ghcr.io/cloudivision",
			DefaultBranch:      "main",
			ServiceAccountName: "sample-runner",
			Isolation: cicdv1alpha1.ProjectIsolation{
				CreateNamespace:   true,
				PodSecurityLevel:  cicdv1alpha1.PodSecurityLevelRestricted,
				NetworkPolicyMode: cicdv1alpha1.NetworkPolicyModeDefaultDeny,
			},
		},
	}
}

func projectRequestFor(project *cicdv1alpha1.Project) ctrl.Request {
	return ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)}
}

func roleAllows(rules []rbacv1.PolicyRule, apiGroup, resource, verb string) bool {
	for _, rule := range rules {
		if contains(rule.APIGroups, apiGroup) && contains(rule.Resources, resource) && contains(rule.Verbs, verb) {
			return true
		}
	}
	return false
}

func assertNoClusterAdminRules(t *testing.T, rules []rbacv1.PolicyRule) {
	t.Helper()
	for _, rule := range rules {
		if contains(rule.Verbs, "*") || contains(rule.Resources, "*") || contains(rule.APIGroups, "*") {
			t.Fatalf("rule grants wildcard permissions: %#v", rule)
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

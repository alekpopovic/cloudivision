package admission

import (
	"context"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDefaults(t *testing.T) {
	project := &cicdv1alpha1.Project{}
	if err := (Defaulter{}).Default(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	if project.Spec.DefaultBranch != "main" {
		t.Fatalf("default branch = %q", project.Spec.DefaultBranch)
	}
	build := &cicdv1alpha1.BuildRun{}
	if err := (Defaulter{}).Default(context.Background(), build); err != nil {
		t.Fatal(err)
	}
	if build.Spec.Executor != cicdv1alpha1.ExecutorTypeJob {
		t.Fatalf("executor = %q", build.Spec.Executor)
	}
	environment := &cicdv1alpha1.Environment{Spec: cicdv1alpha1.EnvironmentSpec{Type: cicdv1alpha1.EnvironmentTypeProduction}}
	if err := (Defaulter{}).Default(context.Background(), environment); err != nil {
		t.Fatal(err)
	}
	if !environment.Spec.RequiresApproval {
		t.Fatal("production approval was not defaulted")
	}
}

func TestValidation(t *testing.T) {
	validator := Validator{}
	_, err := validator.ValidateCreate(context.Background(), &cicdv1alpha1.Repository{Spec: cicdv1alpha1.RepositorySpec{Webhook: cicdv1alpha1.RepositoryWebhook{Enabled: true}}})
	if err == nil || !strings.Contains(err.Error(), "secretRef") {
		t.Fatalf("webhook validation error = %v", err)
	}
	_, err = validator.ValidateCreate(context.Background(), &cicdv1alpha1.PipelineTemplate{Spec: cicdv1alpha1.PipelineTemplateSpec{Steps: []cicdv1alpha1.PipelineStep{{Name: "x", Image: "busybox"}}, Security: cicdv1alpha1.PipelineSecuritySpec{AllowPrivileged: true}}})
	if err == nil || !strings.Contains(err.Error(), AllowPrivilegedAnnotation) {
		t.Fatalf("privileged validation error = %v", err)
	}
	allowed := &cicdv1alpha1.PipelineTemplate{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{AllowPrivilegedAnnotation: "true"}}, Spec: cicdv1alpha1.PipelineTemplateSpec{Steps: []cicdv1alpha1.PipelineStep{{Name: "x", Image: "busybox"}}, Security: cicdv1alpha1.PipelineSecuritySpec{AllowPrivileged: true}}}
	if _, err = validator.ValidateCreate(context.Background(), allowed); err != nil {
		t.Fatalf("explicit privileged opt-in rejected: %v", err)
	}
}

func runtimeScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func TestProductionReleaseRequiresDigest(t *testing.T) {
	scheme := runtimeScheme(t)
	environment := &cicdv1alpha1.Environment{ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "ci"}, Spec: cicdv1alpha1.EnvironmentSpec{Type: cicdv1alpha1.EnvironmentTypeProduction, Policy: cicdv1alpha1.EnvironmentPolicySpec{RequireImageDigest: true}}}
	validator := Validator{Reader: fake.NewClientBuilder().WithScheme(scheme).WithObjects(environment).Build()}
	release := &cicdv1alpha1.Release{ObjectMeta: metav1.ObjectMeta{Namespace: "ci"}, Spec: cicdv1alpha1.ReleaseSpec{EnvironmentRef: "prod", Image: cicdv1alpha1.ImageRef{Repository: "example/image", Tag: "latest"}}}
	if _, err := validator.ValidateCreate(context.Background(), release); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("digest validation error = %v", err)
	}
}

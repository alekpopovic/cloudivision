package job

import (
	"context"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/executor"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureRunAppliesSecurityAndResourceDefaults(t *testing.T) {
	ctx := context.Background()
	scheme := newScheme(t)
	buildRun := testBuildRun()
	project := testProject()
	repository := testRepository()
	template := testPipelineTemplate()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(buildRun).Build()
	jobExecutor := Executor{Client: fakeClient, Scheme: scheme}

	ref, err := jobExecutor.EnsureRun(ctx, executor.EnsureRunRequest{
		BuildRun:   buildRun,
		Project:    project,
		Repository: repository,
		Template:   template,
	})
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}

	job := &batchv1.Job{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, job); err != nil {
		t.Fatalf("get Job error = %v", err)
	}
	podSpec := job.Spec.Template.Spec
	if podSpec.ServiceAccountName != defaultRunnerServiceAccount {
		t.Fatalf("serviceAccountName = %q, want fallback", podSpec.ServiceAccountName)
	}
	if podSpec.AutomountServiceAccountToken == nil || !*podSpec.AutomountServiceAccountToken {
		t.Fatalf("automountServiceAccountToken = %#v, want true", podSpec.AutomountServiceAccountToken)
	}
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.SeccompProfile == nil || podSpec.SecurityContext.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault {
		t.Fatalf("seccompProfile = %#v, want RuntimeDefault", podSpec.SecurityContext)
	}
	if job.Spec.ActiveDeadlineSeconds == nil || *job.Spec.ActiveDeadlineSeconds != defaultActiveDeadlineSeconds {
		t.Fatalf("activeDeadlineSeconds = %#v, want default", job.Spec.ActiveDeadlineSeconds)
	}
	container := podSpec.Containers[0]
	assertQuantity(t, container.Resources.Requests[corev1.ResourceCPU], defaultCPURequest)
	assertQuantity(t, container.Resources.Limits[corev1.ResourceCPU], defaultCPULimit)
	assertQuantity(t, container.Resources.Requests[corev1.ResourceMemory], defaultMemoryRequest)
	assertQuantity(t, container.Resources.Limits[corev1.ResourceMemory], defaultMemoryLimit)
	if container.SecurityContext == nil || container.SecurityContext.Privileged == nil || *container.SecurityContext.Privileged {
		t.Fatalf("privileged = %#v, want false", container.SecurityContext)
	}
}

func TestEnsureRunRejectsPrivilegedTemplateByDefault(t *testing.T) {
	ctx := context.Background()
	scheme := newScheme(t)
	buildRun := testBuildRun()
	template := testPipelineTemplate()
	template.Spec.Security.AllowPrivileged = true
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(buildRun).Build()
	jobExecutor := Executor{Client: fakeClient, Scheme: scheme}

	_, err := jobExecutor.EnsureRun(ctx, executor.EnsureRunRequest{
		BuildRun:   buildRun,
		Project:    testProject(),
		Repository: testRepository(),
		Template:   template,
	})
	if err == nil || !strings.Contains(err.Error(), "privileged") {
		t.Fatalf("EnsureRun() error = %v, want privileged rejection", err)
	}
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}
	return scheme
}

func testBuildRun() *cicdv1alpha1.BuildRun {
	return &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-buildrun", Namespace: "ci"},
		Spec: cicdv1alpha1.BuildRunSpec{
			ProjectRef:          "sample-project",
			RepositoryRef:       "sample-repository",
			PipelineTemplateRef: "sample-template",
			Revision:            "main",
			Image: cicdv1alpha1.ImageRef{
				Repository: "ghcr.io/cloudivision/example",
				Tag:        "main",
			},
		},
	}
}

func testProject() *cicdv1alpha1.Project {
	return &cicdv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-project", Namespace: "ci"},
		Spec: cicdv1alpha1.ProjectSpec{
			DisplayName:     "Sample",
			OwnerTeam:       "platform",
			Namespace:       "ci",
			DefaultRegistry: "ghcr.io/cloudivision",
			DefaultBranch:   "main",
		},
	}
}

func testRepository() *cicdv1alpha1.Repository {
	return &cicdv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-repository", Namespace: "ci"},
		Spec: cicdv1alpha1.RepositorySpec{
			ProjectRef:          "sample-project",
			Provider:            cicdv1alpha1.RepositoryProviderGitHub,
			URL:                 "https://github.com/cloudivision/example.git",
			DefaultBranch:       "main",
			PipelineTemplateRef: "sample-template",
		},
	}
}

func testPipelineTemplate() *cicdv1alpha1.PipelineTemplate {
	return &cicdv1alpha1.PipelineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-template", Namespace: "ci"},
		Spec: cicdv1alpha1.PipelineTemplateSpec{
			Security: cicdv1alpha1.PipelineSecuritySpec{RunAsNonRoot: true},
		},
	}
}

func assertQuantity(t *testing.T, got resource.Quantity, want string) {
	t.Helper()
	wantQuantity := resource.MustParse(want)
	if got.Cmp(wantQuantity) != 0 {
		t.Fatalf("quantity = %s, want %s", got.String(), wantQuantity.String())
	}
}

var _ client.Client

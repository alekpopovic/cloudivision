package job

import (
	"context"
	"errors"
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

func TestBuildJobMountsCosignKeySecretReadOnly(t *testing.T) {
	template := testPipelineTemplate()
	template.Spec.SupplyChain.SignImage = true
	template.Spec.SupplyChain.SignerAdapter = "cosign"
	template.Spec.SupplyChain.SigningKeySecretRef = &cicdv1alpha1.RequiredSecretKeyRef{Name: "cosign-key", Key: "private.key"}
	job := buildJob(testBuildRun(), testProject(), testRepository(), template)
	if len(job.Spec.Template.Spec.Volumes) != 1 || job.Spec.Template.Spec.Volumes[0].Secret == nil {
		t.Fatalf("volumes = %#v", job.Spec.Template.Spec.Volumes)
	}
	secret := job.Spec.Template.Spec.Volumes[0].Secret
	if secret.SecretName != "cosign-key" || len(secret.Items) != 1 || secret.Items[0].Key != "private.key" {
		t.Fatalf("secret projection = %#v", secret)
	}
	mounts := job.Spec.Template.Spec.Containers[0].VolumeMounts
	if len(mounts) != 1 || !mounts[0].ReadOnly || mounts[0].MountPath != "/var/run/secrets/cloudivision-signing" {
		t.Fatalf("volumeMounts = %#v", mounts)
	}
}

func TestBuildJobMountsOptInPVCCache(t *testing.T) {
	t.Setenv("CLOU_DIVISION_CACHE_PVC", "runner-cache")
	t.Setenv("CLOU_DIVISION_CACHE_ROOT", "/cache")
	template := testPipelineTemplate()
	template.Spec.Cache = cicdv1alpha1.PipelineCacheSpec{Enabled: true, Mode: cicdv1alpha1.DependencyCacheModePVC, Paths: []string{"node_modules"}}
	job := buildJob(testBuildRun(), testProject(), testRepository(), template)
	pod := job.Spec.Template.Spec
	if len(pod.Volumes) != 1 || pod.Volumes[0].PersistentVolumeClaim == nil || pod.Volumes[0].PersistentVolumeClaim.ClaimName != "runner-cache" {
		t.Fatalf("volumes = %#v", pod.Volumes)
	}
	container := pod.Containers[0]
	if len(container.VolumeMounts) != 1 || container.VolumeMounts[0].MountPath != "/cache" {
		t.Fatalf("mounts = %#v", container.VolumeMounts)
	}
	if got := envValue(container.Env, "CACHE_BACKEND"); got != "pvc" {
		t.Fatalf("CACHE_BACKEND = %q", got)
	}
}

func TestEnsureRunProjectsOnlyConfiguredRegistrySecret(t *testing.T) {
	ctx := context.Background()
	scheme := newScheme(t)
	buildRun := testBuildRun()
	project := testProject()
	project.Spec.Registry = &cicdv1alpha1.ProjectRegistrySpec{
		Provider:    cicdv1alpha1.RegistryProviderGHCR,
		ImagePrefix: "ghcr.io/cloudivision",
		CredentialSecretRef: &cicdv1alpha1.SecretKeyRef{
			Name: "registry-auth",
			Key:  "config",
		},
	}
	template := testPipelineTemplate()
	template.Spec.Build.Enabled = true
	template.Spec.Build.Push = true
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "registry-auth", Namespace: "ci"},
		Data: map[string][]byte{
			"config": []byte(`{"auths":{"ghcr.io":{"auth":"cm9ib3Q6dG9rZW4="}}}`),
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(buildRun, secret).Build()
	jobExecutor := Executor{Client: fakeClient, Scheme: scheme}

	ref, err := jobExecutor.EnsureRun(ctx, executor.EnsureRunRequest{
		BuildRun: buildRun, Project: project, Repository: testRepository(), Template: template,
	})
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	job := &batchv1.Job{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, job); err != nil {
		t.Fatal(err)
	}
	volumes := job.Spec.Template.Spec.Volumes
	if len(volumes) != 1 || volumes[0].Secret == nil || volumes[0].Secret.SecretName != "registry-auth" {
		t.Fatalf("volumes = %#v", volumes)
	}
	if len(volumes[0].Secret.Items) != 1 || volumes[0].Secret.Items[0].Key != "config" || volumes[0].Secret.Items[0].Path != ".dockerconfigjson" {
		t.Fatalf("registry Secret items = %#v", volumes[0].Secret.Items)
	}
	mounts := job.Spec.Template.Spec.Containers[0].VolumeMounts
	if len(mounts) != 1 || !mounts[0].ReadOnly || mounts[0].MountPath != RegistryCredentialsDir {
		t.Fatalf("mounts = %#v", mounts)
	}
	if got := envValue(job.Spec.Template.Spec.Containers[0].Env, "REGISTRY_PROVIDER"); got != "ghcr" {
		t.Fatalf("REGISTRY_PROVIDER = %q", got)
	}
}

func TestEnsureRunProjectsOnlyRecognizedRegistryKeys(t *testing.T) {
	ctx := context.Background()
	scheme := newScheme(t)
	buildRun, project, template := testBuildRun(), testProject(), testPipelineTemplate()
	project.Spec.Registry = &cicdv1alpha1.ProjectRegistrySpec{Provider: cicdv1alpha1.RegistryProviderGHCR, CredentialSecretRef: &cicdv1alpha1.SecretKeyRef{Name: "registry-auth"}}
	template.Spec.Build.Enabled, template.Spec.Build.Push = true, true
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "registry-auth", Namespace: "ci"}, Data: map[string][]byte{"username": []byte("robot"), "token": []byte("token"), "unrelated": []byte("must-not-mount")}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(buildRun, secret).Build()
	jobExecutor := Executor{Client: fakeClient, Scheme: scheme}
	ref, err := jobExecutor.EnsureRun(ctx, executor.EnsureRunRequest{BuildRun: buildRun, Project: project, Repository: testRepository(), Template: template})
	if err != nil {
		t.Fatal(err)
	}
	job := &batchv1.Job{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, job); err != nil {
		t.Fatal(err)
	}
	items := job.Spec.Template.Spec.Volumes[0].Secret.Items
	if len(items) != 2 {
		t.Fatalf("projected items = %#v", items)
	}
	for _, item := range items {
		if item.Key == "unrelated" {
			t.Fatalf("unrelated key was projected: %#v", items)
		}
	}
}

func TestEnsureRunReportsMissingRegistrySecret(t *testing.T) {
	project := testProject()
	project.Spec.Registry = &cicdv1alpha1.ProjectRegistrySpec{
		Provider:            cicdv1alpha1.RegistryProviderGeneric,
		CredentialSecretRef: &cicdv1alpha1.SecretKeyRef{Name: "missing-auth"},
	}
	template := testPipelineTemplate()
	template.Spec.Build.Enabled = true
	template.Spec.Build.Push = true
	scheme := newScheme(t)
	jobExecutor := Executor{Client: fake.NewClientBuilder().WithScheme(scheme).Build(), Scheme: scheme}

	_, err := jobExecutor.EnsureRun(context.Background(), executor.EnsureRunRequest{
		BuildRun: testBuildRun(), Project: project, Repository: testRepository(), Template: template,
	})
	if !errors.Is(err, ErrRegistryCredentialsMissing) || !strings.Contains(err.Error(), "missing-auth") {
		t.Fatalf("EnsureRun() error = %v", err)
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

func envValue(env []corev1.EnvVar, name string) string {
	for _, item := range env {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}

var _ client.Client

package tekton

import (
	"context"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/executor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureRunCreatesPipelineRunMappingBuildRunInputs(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}
	buildRun := testBuildRun()
	project := testProject()
	repository := testRepository()
	template := testPipelineTemplate()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(buildRun).Build()
	tektonExecutor := Executor{Client: fakeClient, Scheme: scheme}

	ref, err := tektonExecutor.EnsureRun(ctx, executor.EnsureRunRequest{
		BuildRun:   buildRun,
		Project:    project,
		Repository: repository,
		Template:   template,
	})
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	if ref.Kind != "PipelineRun" {
		t.Fatalf("kind = %q, want PipelineRun", ref.Kind)
	}

	run := pipelineRun()
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, run); err != nil {
		t.Fatalf("get PipelineRun error = %v", err)
	}
	params, _, _ := unstructuredSlice(run.Object, "spec", "params")
	if !containsParam(params, "repository-url", repository.Spec.URL) {
		t.Fatalf("params = %#v, want repository-url", params)
	}
	if !containsParam(params, "revision", buildRun.Spec.Revision) {
		t.Fatalf("params = %#v, want revision", params)
	}
	tasks, _, _ := unstructuredSlice(run.Object, "spec", "pipelineSpec", "tasks")
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
}

func testBuildRun() *cicdv1alpha1.BuildRun {
	return &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-buildrun", Namespace: "ci"},
		Spec: cicdv1alpha1.BuildRunSpec{
			ProjectRef:          "sample-project",
			RepositoryRef:       "sample-repository",
			PipelineTemplateRef: "sample-template",
			Revision:            "abc123",
			Branch:              "main",
			Image: cicdv1alpha1.ImageRef{
				Repository: "ghcr.io/cloudivision/example",
				Tag:        "main",
			},
			Executor: cicdv1alpha1.ExecutorTypeTekton,
		},
	}
}

func testProject() *cicdv1alpha1.Project {
	return &cicdv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-project", Namespace: "ci"},
		Spec: cicdv1alpha1.ProjectSpec{
			DisplayName:     "Sample Project",
			OwnerTeam:       "platform",
			Namespace:       "sample",
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
			Steps: []cicdv1alpha1.PipelineStep{
				{Name: "test", Image: "golang:1.26", Command: []string{"go"}, Args: []string{"test", "./..."}},
			},
			Build: cicdv1alpha1.PipelineBuildSpec{
				Enabled:    true,
				ContextDir: ".",
				Dockerfile: "Dockerfile",
				Builder:    cicdv1alpha1.BuildBuilderBuildKit,
			},
		},
	}
}

func unstructuredSlice(root map[string]any, fields ...string) ([]any, bool, error) {
	current := root
	for _, field := range fields[:len(fields)-1] {
		next, ok := current[field].(map[string]any)
		if !ok {
			return nil, false, nil
		}
		current = next
	}
	values, ok := current[fields[len(fields)-1]].([]any)
	return values, ok, nil
}

func containsParam(params []any, name, value string) bool {
	for _, raw := range params {
		param, ok := raw.(map[string]any)
		if ok && param["name"] == name && param["value"] == value {
			return true
		}
	}
	return false
}

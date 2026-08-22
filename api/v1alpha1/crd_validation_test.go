package v1alpha1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPendingBuildRunStatusOmitsArtifactImage(t *testing.T) {
	data, err := json.Marshal(BuildRunStatus{Phase: BuildRunPhasePending})
	if err != nil {
		t.Fatalf("marshal pending BuildRun status: %v", err)
	}
	if strings.Contains(string(data), `"image"`) {
		t.Fatalf("pending status contains an empty validated image: %s", data)
	}
}

func TestProjectWithoutRegistryOmitsOptionalRegistryConfig(t *testing.T) {
	data, err := json.Marshal(ProjectSpec{})
	if err != nil {
		t.Fatalf("marshal Project spec: %v", err)
	}
	if strings.Contains(string(data), `"registry"`) {
		t.Fatalf("empty Project spec contains registry config: %s", data)
	}
}

func TestGeneratedCRDValidationContract(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"cicd.cloudivision.io_projects.yaml": {
			"default: main",
			"- namespace",
		},
		"cicd.cloudivision.io_repositories.yaml": {
			"webhook.secretRef is required when webhook.enabled is true",
			"- projectRef",
			"- url",
		},
		"cicd.cloudivision.io_pipelinetemplates.yaml": {
			"at least one step or an enabled image build is required",
			"x-kubernetes-list-type: map",
			"x-kubernetes-list-map-keys:",
			"minItems: 1",
			"default: Dockerfile",
		},
		"cicd.cloudivision.io_buildruns.yaml": {
			"gitOps.strategy is required when gitOps.enabled is true",
			"gitOps.environmentRef is required when gitOps.enabled is",
			"gitOps.repoURL is required when gitOps.enabled is true",
			"default: job",
			"- pipelineTemplateRef",
			"- projectRef",
			"- repositoryRef",
		},
		"cicd.cloudivision.io_environments.yaml": {
			"production environments must require approval",
			"- production",
		},
		"cicd.cloudivision.io_releases.yaml": {
			"release image must include a tag or digest",
			"- environmentRef",
			"- buildRunRef",
			"- projectRef",
		},
	}

	for filename, required := range cases {
		filename, required := filename, required
		t.Run(filename, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join("..", "..", "config", "crd", "bases", filename)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read generated CRD %s: %v", path, err)
			}
			manifest := string(data)
			for _, fragment := range required {
				if !strings.Contains(manifest, fragment) {
					t.Errorf("generated CRD %s does not contain %q", filename, fragment)
				}
			}
		})
	}
}

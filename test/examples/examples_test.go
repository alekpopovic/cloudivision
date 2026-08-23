package examples_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func TestEveryExampleHasTypedResourcesAndSafePipeline(t *testing.T) {
	manifests, err := filepath.Glob(filepath.Join("..", "..", "examples", "*", "cloudivision.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifests) != 8 {
		t.Fatalf("found %d example manifests, want 8", len(manifests))
	}
	for _, manifest := range manifests {
		manifest := manifest
		t.Run(filepath.Base(filepath.Dir(manifest)), func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(filepath.Dir(manifest), "README.md")); err != nil {
				t.Fatalf("README: %v", err)
			}
			file, err := os.Open(manifest)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			decoder := utilyaml.NewYAMLOrJSONDecoder(file, 4096)
			seen := map[string]bool{}
			for {
				var object unstructured.Unstructured
				if err := decoder.Decode(&object); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("decode: %v", err)
				}
				seen[object.GetKind()] = true
				switch object.GetKind() {
				case "PipelineTemplate":
					var value cicdv1alpha1.PipelineTemplate
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &value); err != nil {
						t.Fatalf("typed PipelineTemplate: %v", err)
					}
					if len(value.Spec.Steps) == 0 || value.Spec.Resources.TimeoutSeconds == 0 {
						t.Fatal("pipeline requires a step and an explicit timeout")
					}
					if value.Spec.Security.AllowPrivileged || !value.Spec.Security.RunAsNonRoot {
						t.Fatal("pipeline must be non-privileged and run as non-root")
					}
				case "Repository":
					var value cicdv1alpha1.Repository
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &value); err != nil {
						t.Fatalf("typed Repository: %v", err)
					}
				case "BuildRun":
					var value cicdv1alpha1.BuildRun
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &value); err != nil {
						t.Fatalf("typed BuildRun: %v", err)
					}
				default:
					t.Fatalf("unexpected kind %q", object.GetKind())
				}
			}
			for _, kind := range []string{"PipelineTemplate", "Repository", "BuildRun"} {
				if !seen[kind] {
					t.Errorf("missing %s", kind)
				}
			}
		})
	}
}

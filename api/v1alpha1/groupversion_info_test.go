package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestAddToSchemeRegistersAPITypes(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme() error = %v", err)
	}

	for _, kind := range []string{
		"Project",
		"Repository",
		"PipelineTemplate",
		"BuildRun",
		"Environment",
		"Release",
	} {
		gvk := schema.GroupVersionKind{
			Group:   GroupVersion.Group,
			Version: GroupVersion.Version,
			Kind:    kind,
		}
		if _, err := scheme.New(gvk); err != nil {
			t.Fatalf("scheme.New(%s) error = %v", gvk.String(), err)
		}
	}
}

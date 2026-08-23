package catalog

import "testing"

func TestPipelineTemplateCatalogLoads(t *testing.T) {
	items := PipelineTemplates()
	if len(items) != 11 {
		t.Fatalf("templates = %d, want 11", len(items))
	}
	for _, name := range []string{"go", "node-npm", "angular", "dockerfile"} {
		item, ok := FindPipelineTemplate(name)
		if !ok || item.Version == "" {
			t.Fatalf("template %q missing or unversioned", name)
		}
	}
}

func TestFindPipelineTemplateRejectsUnknownName(t *testing.T) {
	if _, ok := FindPipelineTemplate("unknown"); ok {
		t.Fatal("unknown template found")
	}
}

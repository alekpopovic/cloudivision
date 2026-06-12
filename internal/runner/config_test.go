package runner

import "testing"

func TestConfigFromEnv(t *testing.T) {
	env := map[string]string{
		"BUILD_RUN_NAME":         "build-1",
		"BUILD_RUN_NAMESPACE":    "ci",
		"PROJECT_NAME":           "project",
		"REPOSITORY_URL":         "https://example.com/repo.git",
		"REVISION":               "main",
		"BRANCH":                 "main",
		"PIPELINE_TEMPLATE_NAME": "template",
		"IMAGE_REPOSITORY":       "ghcr.io/cloudivision/app",
		"IMAGE_TAG":              "main",
		"GITOPS_ENABLED":         "true",
	}
	cfg, err := ConfigFromEnv(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("ConfigFromEnv() error = %v", err)
	}
	if !cfg.GitOpsEnabled {
		t.Fatal("GitOpsEnabled = false, want true")
	}
	if cfg.BuildRunName != "build-1" {
		t.Fatalf("BuildRunName = %q", cfg.BuildRunName)
	}
}

func TestConfigFromEnvRequiresBuildRunName(t *testing.T) {
	_, err := ConfigFromEnv(func(key string) string {
		values := map[string]string{
			"BUILD_RUN_NAMESPACE":    "ci",
			"PROJECT_NAME":           "project",
			"REPOSITORY_URL":         "https://example.com/repo.git",
			"REVISION":               "main",
			"PIPELINE_TEMPLATE_NAME": "template",
			"IMAGE_REPOSITORY":       "ghcr.io/cloudivision/app",
		}
		return values[key]
	})
	if err == nil {
		t.Fatal("ConfigFromEnv() error = nil, want error")
	}
}

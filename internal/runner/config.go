package runner

import (
	"fmt"
	"strconv"
)

type Config struct {
	BuildRunName         string
	BuildRunNamespace    string
	ProjectName          string
	RepositoryURL        string
	Revision             string
	Branch               string
	PipelineTemplateName string
	ImageRepository      string
	ImageTag             string
	GitOpsEnabled        bool
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		BuildRunName:         getenv("BUILD_RUN_NAME"),
		BuildRunNamespace:    getenv("BUILD_RUN_NAMESPACE"),
		ProjectName:          getenv("PROJECT_NAME"),
		RepositoryURL:        getenv("REPOSITORY_URL"),
		Revision:             getenv("REVISION"),
		Branch:               getenv("BRANCH"),
		PipelineTemplateName: getenv("PIPELINE_TEMPLATE_NAME"),
		ImageRepository:      getenv("IMAGE_REPOSITORY"),
		ImageTag:             getenv("IMAGE_TAG"),
	}
	if value := getenv("GITOPS_ENABLED"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse GITOPS_ENABLED: %w", err)
		}
		cfg.GitOpsEnabled = parsed
	}
	required := map[string]string{
		"BUILD_RUN_NAME":         cfg.BuildRunName,
		"BUILD_RUN_NAMESPACE":    cfg.BuildRunNamespace,
		"PROJECT_NAME":           cfg.ProjectName,
		"REPOSITORY_URL":         cfg.RepositoryURL,
		"PIPELINE_TEMPLATE_NAME": cfg.PipelineTemplateName,
		"IMAGE_REPOSITORY":       cfg.ImageRepository,
	}
	for name, value := range required {
		if value == "" {
			return Config{}, fmt.Errorf("%s is required", name)
		}
	}
	if cfg.Revision == "" && cfg.Branch == "" {
		return Config{}, fmt.Errorf("REVISION or BRANCH is required")
	}
	return cfg, nil
}

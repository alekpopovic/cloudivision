package runner

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
)

type Config struct {
	BuildRunName           string
	BuildRunNamespace      string
	ProjectName            string
	RepositoryURL          string
	Revision               string
	Branch                 string
	PipelineTemplateName   string
	ImageRepository        string
	ImageTag               string
	RegistryProvider       string
	RegistryImagePrefix    string
	RegistryCredentialsDir string
	GitOpsEnabled          bool
	LogBackend             string
	LogRoot                string
	ArtifactBackend        string
	ArtifactRoot           string
	MaxArtifactsBytes      int64
	MaxLogBytes            int64
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		BuildRunName:           getenv("BUILD_RUN_NAME"),
		BuildRunNamespace:      getenv("BUILD_RUN_NAMESPACE"),
		ProjectName:            getenv("PROJECT_NAME"),
		RepositoryURL:          getenv("REPOSITORY_URL"),
		Revision:               getenv("REVISION"),
		Branch:                 getenv("BRANCH"),
		PipelineTemplateName:   getenv("PIPELINE_TEMPLATE_NAME"),
		ImageRepository:        getenv("IMAGE_REPOSITORY"),
		ImageTag:               getenv("IMAGE_TAG"),
		RegistryProvider:       getenv("REGISTRY_PROVIDER"),
		RegistryImagePrefix:    getenv("REGISTRY_IMAGE_PREFIX"),
		RegistryCredentialsDir: getenv("REGISTRY_CREDENTIALS_DIR"),
		LogBackend:             getenv("LOG_BACKEND"),
		LogRoot:                getenv("LOG_ROOT"),
		ArtifactBackend:        getenv("ARTIFACT_BACKEND"),
		ArtifactRoot:           getenv("ARTIFACT_ROOT"),
	}
	var err error
	if value := getenv("PROJECT_MAX_ARTIFACTS_SIZE"); value != "" {
		cfg.MaxArtifactsBytes, err = byteQuantity(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse PROJECT_MAX_ARTIFACTS_SIZE: %w", err)
		}
	}
	if value := getenv("PROJECT_MAX_LOG_SIZE"); value != "" {
		cfg.MaxLogBytes, err = byteQuantity(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse PROJECT_MAX_LOG_SIZE: %w", err)
		}
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

func byteQuantity(value string) (int64, error) {
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0, err
	}
	bytes := quantity.Value()
	if bytes <= 0 {
		return 0, fmt.Errorf("must be greater than zero")
	}
	return bytes, nil
}

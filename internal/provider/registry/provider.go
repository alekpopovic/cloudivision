package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	coreprovider "github.com/cloudivision/cloudivision/internal/provider"
)

func Generic() *Adapter {
	return &Adapter{ProviderName: "generic", Implemented: true}
}

func GHCR() *Adapter {
	return &Adapter{ProviderName: "ghcr", DefaultRegistry: "ghcr.io", Implemented: true}
}

func GitLab() *Adapter {
	return &Adapter{ProviderName: "gitlab", DefaultRegistry: "registry.gitlab.com", Implemented: true}
}

func Harbor() *Adapter {
	return &Adapter{ProviderName: "harbor", Implemented: true}
}

func ECR() *Adapter { return &Adapter{ProviderName: "ecr"} }
func GCR() *Adapter { return &Adapter{ProviderName: "gcr"} }
func ACR() *Adapter { return &Adapter{ProviderName: "acr"} }

func New(name string) (RegistryProvider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "generic":
		return Generic(), nil
	case "ghcr":
		return GHCR(), nil
	case "gitlab":
		return GitLab(), nil
	case "harbor":
		return Harbor(), nil
	case "ecr":
		return ECR(), nil
	case "gcr":
		return GCR(), nil
	case "acr":
		return ACR(), nil
	default:
		return nil, fmt.Errorf("registry provider %q: %w", name, ErrUnsupportedProvider)
	}
}

func (p *Adapter) Name() string { return p.ProviderName }
func (p *Adapter) Type() string { return "registry" }
func (p *Adapter) Capabilities() []coreprovider.Capability {
	capabilities := []coreprovider.Capability{
		{Name: "credentials", Description: "Create scoped Docker registry authentication configuration"},
		{Name: "resolve-image", Description: "Resolve provider image prefixes and immutable references"},
	}
	if p.Implemented {
		capabilities = append(capabilities, coreprovider.Capability{Name: "digest", Description: "Read OCI manifest digests"})
	} else {
		capabilities = append(capabilities, coreprovider.Capability{Name: "skeleton", Description: "Cloud credential exchange is reserved for a future adapter"})
	}
	return capabilities
}

func (p *Adapter) HealthCheck(context.Context) coreprovider.ProviderHealth {
	health := coreprovider.ProviderHealth{CheckedAt: time.Now().UTC()}
	if p == nil || p.ProviderName == "" {
		health.Message = "registry provider is not configured"
		return health
	}
	if !p.Implemented {
		health.Message = fmt.Sprintf("%s cloud credential exchange is a skeleton", p.ProviderName)
		return health
	}
	health.Healthy = true
	health.Message = fmt.Sprintf("%s registry provider is available", p.ProviderName)
	return health
}

func (p *Adapter) Login(_ context.Context, req LoginRequest) (*LoginResult, error) {
	if p == nil || !p.Implemented {
		name := "unknown"
		if p != nil {
			name = p.ProviderName
		}
		return nil, fmt.Errorf("registry provider %q: %w", name, ErrUnsupportedProvider)
	}
	registry := strings.TrimSpace(req.Registry)
	if registry == "" {
		registry = p.DefaultRegistry
	}
	registry = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(registry, "https://"), "http://"), "/")
	if registry == "" {
		return nil, fmt.Errorf("registry host is required: %w", ErrImageInvalid)
	}
	if req.Credential.Empty() {
		return nil, ErrCredentialsMissing
	}
	config, err := dockerConfigJSON(registry, req.Credential)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Registry: registry, DockerConfigJSON: config}, nil
}

func dockerConfigJSON(registry string, credential Credential) ([]byte, error) {
	if len(credential.DockerConfigJSON) > 0 {
		var config struct {
			Auths map[string]json.RawMessage `json:"auths"`
		}
		if err := json.Unmarshal(credential.DockerConfigJSON, &config); err != nil {
			return nil, fmt.Errorf("parse dockerconfigjson: %w: %w", err, ErrCredentialsInvalid)
		}
		if len(config.Auths) == 0 {
			return nil, fmt.Errorf("dockerconfigjson contains no auths: %w", ErrCredentialsInvalid)
		}
		return append([]byte(nil), credential.DockerConfigJSON...), nil
	}
	secret := credential.Password
	if credential.Token != "" {
		secret = credential.Token
	}
	if secret == "" {
		return nil, fmt.Errorf("password or token is required: %w", ErrCredentialsInvalid)
	}
	username := credential.Username
	if username == "" {
		username = "oauth2"
	}
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + secret))
	config := map[string]any{
		"auths": map[string]any{
			registry: map[string]string{
				"auth":     auth,
				"username": username,
				"password": secret,
			},
		},
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("encode docker config: %w", err)
	}
	return data, nil
}

var _ coreprovider.Provider = (*Adapter)(nil)
var _ RegistryProvider = (*Adapter)(nil)

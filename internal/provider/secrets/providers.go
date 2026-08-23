package secrets

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/cloudivision/cloudivision/internal/provider"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrSecretMissing        = errors.New("secret is missing")
	ErrSecretKeyMissing     = errors.New("secret key is missing")
	ErrNamespaceForbidden   = errors.New("secret namespace is forbidden")
	ErrExplicitKeysRequired = errors.New("explicit secret keys are required")
	ErrUnsupported          = errors.New("secret provider is unsupported")
)

type SecretRef struct {
	Namespace    string
	Name         string
	Keys         []string
	OptionalKeys []string
	AllowAll     bool
}

type ResolvedSecret struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Keys      []string          `json:"keys"`
	Values    map[string][]byte `json:"-"`
}

func (ResolvedSecret) String() string { return "[REDACTED SECRET]" }

type SecretProvider interface {
	Resolve(ctx context.Context, ref SecretRef) (*ResolvedSecret, error)
	HealthCheck(ctx context.Context) provider.ProviderHealth
}

type KubernetesProvider struct {
	Client            client.Client
	AllowedNamespaces []string
	AllowAllKeys      bool
}

func (p KubernetesProvider) Name() string { return "kubernetes" }
func (p KubernetesProvider) Type() string { return "secrets" }
func (p KubernetesProvider) Capabilities() []provider.Capability {
	return []provider.Capability{{Name: "resolve-selected-keys", Description: "Resolve explicit keys from a namespaced Kubernetes Secret"}}
}
func (p KubernetesProvider) HealthCheck(context.Context) provider.ProviderHealth {
	healthy := p.Client != nil
	message := "Kubernetes Secret client is configured"
	if !healthy {
		message = "Kubernetes Secret client is configured at runtime"
		healthy = true
	}
	return provider.ProviderHealth{Healthy: healthy, Message: message, CheckedAt: time.Now().UTC()}
}
func (p KubernetesProvider) Resolve(ctx context.Context, ref SecretRef) (*ResolvedSecret, error) {
	if p.Client == nil {
		return nil, fmt.Errorf("resolve Kubernetes Secret: %w", ErrUnsupported)
	}
	if ref.Namespace == "" || !slices.Contains(p.AllowedNamespaces, ref.Namespace) {
		return nil, ErrNamespaceForbidden
	}
	if ref.Name == "" {
		return nil, fmt.Errorf("secret name is required: %w", ErrSecretMissing)
	}
	if len(ref.Keys) == 0 && len(ref.OptionalKeys) == 0 && !(ref.AllowAll && p.AllowAllKeys) {
		return nil, ErrExplicitKeysRequired
	}
	secret := &corev1.Secret{}
	if err := p.Client.Get(ctx, client.ObjectKey{Namespace: ref.Namespace, Name: ref.Name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrSecretMissing
		}
		return nil, fmt.Errorf("resolve Kubernetes Secret metadata: %w", err)
	}
	keys := append([]string(nil), ref.Keys...)
	if len(keys) == 0 && ref.AllowAll && p.AllowAllKeys {
		for key := range secret.Data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
	}
	values := make(map[string][]byte, len(keys))
	for _, key := range keys {
		value, ok := secret.Data[key]
		if !ok {
			return nil, fmt.Errorf("key %q: %w", key, ErrSecretKeyMissing)
		}
		values[key] = append([]byte(nil), value...)
	}
	for _, key := range ref.OptionalKeys {
		if value, ok := secret.Data[key]; ok {
			values[key] = append([]byte(nil), value...)
			keys = append(keys, key)
		}
	}
	return &ResolvedSecret{Namespace: ref.Namespace, Name: ref.Name, Keys: keys, Values: values}, nil
}

func Kubernetes(clients ...client.Client) provider.Provider {
	var configured client.Client
	if len(clients) > 0 {
		configured = clients[0]
	}
	return KubernetesProvider{Client: configured}
}

type ExternalSecretsProvider struct{ Client client.Client }

func (p ExternalSecretsProvider) Name() string { return "external-secrets" }
func (p ExternalSecretsProvider) Type() string { return "secrets" }
func (p ExternalSecretsProvider) Capabilities() []provider.Capability {
	return []provider.Capability{{Name: "detect-crd", Description: "Detect the ExternalSecret CRD; value resolution remains Kubernetes Secret based"}}
}
func (p ExternalSecretsProvider) Resolve(context.Context, SecretRef) (*ResolvedSecret, error) {
	return nil, ErrUnsupported
}
func (p ExternalSecretsProvider) HealthCheck(ctx context.Context) provider.ProviderHealth {
	health := provider.ProviderHealth{CheckedAt: time.Now().UTC(), Message: "ExternalSecret CRD is not installed"}
	if p.Client == nil {
		return health
	}
	crd := &unstructured.Unstructured{}
	crd.SetGroupVersionKind(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})
	if err := p.Client.Get(ctx, client.ObjectKey{Name: "externalsecrets.external-secrets.io"}, crd); err == nil {
		health.Healthy = true
		health.Message = "ExternalSecret CRD is installed"
	}
	return health
}
func ExternalSecrets(client client.Client) provider.Provider {
	return ExternalSecretsProvider{Client: client}
}

type VaultConfig struct {
	Address    string
	AuthMethod string
	Mount      string
}
type VaultProvider struct{ Config VaultConfig }

func (p VaultProvider) Name() string { return "vault" }
func (p VaultProvider) Type() string { return "secrets" }
func (p VaultProvider) Capabilities() []provider.Capability {
	return []provider.Capability{{Name: "configuration-skeleton", Description: "Vault configuration skeleton without value delivery"}}
}
func (p VaultProvider) Resolve(context.Context, SecretRef) (*ResolvedSecret, error) {
	return nil, ErrUnsupported
}
func (p VaultProvider) HealthCheck(context.Context) provider.ProviderHealth {
	configured := p.Config.Address != "" && p.Config.AuthMethod != ""
	message := "Vault provider is not configured"
	if configured {
		message = "Vault configuration is present; resolver implementation is pending"
	}
	return provider.ProviderHealth{Healthy: false, Message: message, CheckedAt: time.Now().UTC()}
}
func Vault(config VaultConfig) provider.Provider { return VaultProvider{Config: config} }

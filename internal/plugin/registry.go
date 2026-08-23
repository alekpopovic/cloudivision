// Package plugin provides the in-process plugin registry. External processes,
// dynamic loading, and arbitrary plugin code are intentionally unsupported.
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/cloudivision/cloudivision/internal/provider"
)

type Type string

const (
	TypeGit          Type = "git"
	TypeRegistry     Type = "registry"
	TypeBuild        Type = "build"
	TypeGitOps       Type = "gitops"
	TypeNotification Type = "notification"
	TypePolicy       Type = "policy"
	TypeSupplyChain  Type = "supply-chain"
)

var (
	ErrInvalid         = errors.New("invalid plugin")
	ErrDuplicate       = errors.New("plugin already registered")
	ErrNotFound        = errors.New("plugin not found")
	ErrUnsupportedType = errors.New("unsupported plugin type")
)

type Metadata struct {
	Name                string                `json:"name"`
	Type                Type                  `json:"type"`
	Version             string                `json:"version"`
	Capabilities        []provider.Capability `json:"capabilities"`
	ConfigSchema        json.RawMessage       `json:"configSchema"`
	Configured          bool                  `json:"configured"`
	ConfigurationStatus string                `json:"configurationStatus"`
	DocsURL             string                `json:"docsUrl,omitempty"`
}
type HealthResult struct {
	Metadata Metadata                `json:"metadata"`
	Health   provider.ProviderHealth `json:"health"`
}
type Plugin interface {
	Metadata() Metadata
	HealthCheck(context.Context) provider.ProviderHealth
}
type Static struct {
	Info   Metadata
	Health provider.ProviderHealth
}

func (s Static) Metadata() Metadata { return cloneMetadata(s.Info) }
func (s Static) HealthCheck(context.Context) provider.ProviderHealth {
	health := s.Health
	if health.CheckedAt.IsZero() {
		health.CheckedAt = time.Now().UTC()
	}
	return health
}

type providerAdapter struct {
	provider provider.Provider
	info     Metadata
}

func FromProvider(value provider.Provider) Plugin {
	pluginType := Type(value.Type())
	if pluginType == "notifications" {
		pluginType = TypeNotification
	}
	configured := true
	status := "built in"
	if value.Type() == "build" && value.Name() == "remote-agent" {
		configured = false
		status = "experimental and disabled"
	}
	return providerAdapter{provider: value, info: Metadata{Name: value.Name(), Type: pluginType, Version: "builtin", Capabilities: value.Capabilities(), ConfigSchema: json.RawMessage(`{"type":"object","additionalProperties":false}`), Configured: configured, ConfigurationStatus: status, DocsURL: "/plugins/"}}
}
func (a providerAdapter) Metadata() Metadata { return cloneMetadata(a.info) }
func (a providerAdapter) HealthCheck(ctx context.Context) provider.ProviderHealth {
	return a.provider.HealthCheck(ctx)
}

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

func NewRegistry() *Registry { return &Registry{plugins: map[string]Plugin{}} }
func (r *Registry) Register(value Plugin) error {
	if value == nil {
		return ErrInvalid
	}
	info := value.Metadata()
	if info.Name == "" || info.Version == "" {
		return fmt.Errorf("%s/%s: %w", info.Type, info.Name, ErrInvalid)
	}
	if !supported(info.Type) {
		return fmt.Errorf("%s/%s: %w", info.Type, info.Name, ErrUnsupportedType)
	}
	if len(info.ConfigSchema) == 0 || !json.Valid(info.ConfigSchema) {
		return fmt.Errorf("%s/%s config schema: %w", info.Type, info.Name, ErrInvalid)
	}
	key := string(info.Type) + "/" + info.Name
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plugins[key]; ok {
		return fmt.Errorf("%s: %w", key, ErrDuplicate)
	}
	r.plugins[key] = value
	return nil
}
func (r *Registry) Get(pluginType Type, name string) (Plugin, error) {
	if r == nil {
		return nil, ErrNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.plugins[string(pluginType)+"/"+name]
	if !ok {
		return nil, fmt.Errorf("%s/%s: %w", pluginType, name, ErrNotFound)
	}
	return value, nil
}
func (r *Registry) List() []Metadata {
	if r == nil {
		return []Metadata{}
	}
	r.mu.RLock()
	out := make([]Metadata, 0, len(r.plugins))
	for _, value := range r.plugins {
		out = append(out, value.Metadata())
	}
	r.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return string(out[i].Type)+"/"+out[i].Name < string(out[j].Type)+"/"+out[j].Name })
	return out
}
func (r *Registry) Health(ctx context.Context) []HealthResult {
	metadata := r.List()
	out := make([]HealthResult, 0, len(metadata))
	for _, info := range metadata {
		value, _ := r.Get(info.Type, info.Name)
		health := value.HealthCheck(ctx)
		if health.CheckedAt.IsZero() {
			health.CheckedAt = time.Now().UTC()
		}
		out = append(out, HealthResult{Metadata: info, Health: health})
	}
	return out
}
func supported(value Type) bool {
	switch value {
	case TypeGit, TypeRegistry, TypeBuild, TypeGitOps, TypeNotification, TypePolicy, TypeSupplyChain:
		return true
	}
	return false
}
func cloneMetadata(value Metadata) Metadata {
	value.Capabilities = append([]provider.Capability(nil), value.Capabilities...)
	value.ConfigSchema = append(json.RawMessage(nil), value.ConfigSchema...)
	return value
}

package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrInvalidProvider   = errors.New("invalid provider")
	ErrDuplicateProvider = errors.New("provider already registered")
	ErrProviderNotFound  = errors.New("provider not found")
)

type Provider interface {
	Name() string
	Type() string
	HealthCheck(ctx context.Context) ProviderHealth
	Capabilities() []Capability
}

type ProviderHealth struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checkedAt"`
}

type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Summary struct {
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	Capabilities []Capability `json:"capabilities"`
}

type HealthResult struct {
	Summary
	Health ProviderHealth `json:"health"`
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry { return &Registry{providers: map[string]Provider{}} }

func (r *Registry) Register(p Provider) error {
	if p == nil || p.Type() == "" || p.Name() == "" {
		return ErrInvalidProvider
	}
	key := providerKey(p.Type(), p.Name())
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[key]; exists {
		return fmt.Errorf("%s: %w", key, ErrDuplicateProvider)
	}
	r.providers[key] = p
	return nil
}

func (r *Registry) Get(providerType, name string) (Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("%s: %w", providerKey(providerType, name), ErrProviderNotFound)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[providerKey(providerType, name)]
	if !ok {
		return nil, fmt.Errorf("provider type=%q name=%q: %w", providerType, name, ErrProviderNotFound)
	}
	return p, nil
}

func (r *Registry) List() []Provider {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	r.mu.RUnlock()
	sort.Slice(providers, func(i, j int) bool {
		return providerKey(providers[i].Type(), providers[i].Name()) < providerKey(providers[j].Type(), providers[j].Name())
	})
	return providers
}

func (r *Registry) Summaries() []Summary {
	providers := r.List()
	result := make([]Summary, 0, len(providers))
	for _, p := range providers {
		result = append(result, Summary{Name: p.Name(), Type: p.Type(), Capabilities: append([]Capability(nil), p.Capabilities()...)})
	}
	return result
}

func (r *Registry) HealthCheckAll(ctx context.Context) []HealthResult {
	providers := r.List()
	results := make([]HealthResult, len(providers))
	var wg sync.WaitGroup
	for i, p := range providers {
		wg.Add(1)
		go func(index int, current Provider) {
			defer wg.Done()
			health := current.HealthCheck(ctx)
			if health.CheckedAt.IsZero() {
				health.CheckedAt = time.Now().UTC()
			}
			results[index] = HealthResult{Summary: Summary{Name: current.Name(), Type: current.Type(), Capabilities: append([]Capability(nil), current.Capabilities()...)}, Health: health}
		}(i, p)
	}
	wg.Wait()
	return results
}

func providerKey(providerType, name string) string { return providerType + "/" + name }

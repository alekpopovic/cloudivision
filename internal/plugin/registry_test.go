package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudivision/cloudivision/internal/provider"
)

func TestRegistryRegistrationDuplicateHealthAndUnsupported(t *testing.T) {
	registry := NewRegistry()
	value := Static{Info: Metadata{Name: "default", Type: TypePolicy, Version: "1.0.0", Configured: true, ConfigSchema: []byte(`{"type":"object"}`)}, Health: provider.ProviderHealth{Healthy: true, Message: "ready"}}
	if err := registry.Register(value); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(value); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate error=%v", err)
	}
	if got, err := registry.Get(TypePolicy, "default"); err != nil || got.Metadata().Name != "default" {
		t.Fatalf("get=%#v %v", got, err)
	}
	health := registry.Health(context.Background())
	if len(health) != 1 || !health[0].Health.Healthy || health[0].Health.CheckedAt.IsZero() {
		t.Fatalf("health=%#v", health)
	}
	bad := Static{Info: Metadata{Name: "bad", Type: "external", Version: "1"}}
	if err := registry.Register(bad); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("unsupported error=%v", err)
	}
	if _, err := registry.Get(TypeGit, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error=%v", err)
	}
}

func TestProviderAdapterNormalizesNotificationType(t *testing.T) {
	adapted := FromProvider(provider.Static{ProviderName: "webhook", ProviderType: "notifications", Healthy: true})
	if adapted.Metadata().Type != TypeNotification {
		t.Fatalf("type=%q", adapted.Metadata().Type)
	}
}

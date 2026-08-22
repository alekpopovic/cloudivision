package provider

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRegisterGetListAndDuplicate(t *testing.T) {
	registry := NewRegistry()
	git := Static{ProviderName: "generic", ProviderType: "git", Healthy: true, Message: "ok"}
	if err := registry.Register(git); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(git); !errors.Is(err, ErrDuplicateProvider) {
		t.Fatalf("duplicate Register() error = %v", err)
	}
	got, err := registry.Get("git", "generic")
	if err != nil || got.Name() != "generic" {
		t.Fatalf("Get() = %#v, %v", got, err)
	}
	if _, err := registry.Get("git", "missing"); !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("missing Get() error = %v", err)
	}
	if list := registry.List(); len(list) != 1 {
		t.Fatalf("List() = %#v", list)
	}
}

func TestRegistryHealthCheckAllPreservesProviderMetadata(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Static{ProviderName: "noop", ProviderType: "notifications", Healthy: true, Message: "disabled", Features: []Capability{{Name: "discard", Description: "discard messages"}}}); err != nil {
		t.Fatal(err)
	}
	results := registry.HealthCheckAll(context.Background())
	if len(results) != 1 || !results[0].Health.Healthy || results[0].Health.CheckedAt.IsZero() {
		t.Fatalf("HealthCheckAll() = %#v", results)
	}
	if results[0].Type != "notifications" || len(results[0].Capabilities) != 1 {
		t.Fatalf("metadata = %#v", results[0])
	}
}

func TestRegistryRejectsInvalidProviderWithoutPanic(t *testing.T) {
	if err := NewRegistry().Register(nil); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("Register(nil) error = %v", err)
	}
}

package cache

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScopeKeyIsProjectAndRepositoryIsolated(t *testing.T) {
	first, err := ScopeKey("project-a", "repo", "deps")
	if err != nil {
		t.Fatal(err)
	}
	otherProject, _ := ScopeKey("project-b", "repo", "deps")
	otherRepository, _ := ScopeKey("project-a", "other", "deps")
	if first == otherProject || first == otherRepository {
		t.Fatalf("scope collision: %q %q %q", first, otherProject, otherRepository)
	}
}

func TestRegistryRefIsScoped(t *testing.T) {
	first, err := RegistryRef("ghcr.io/acme/app", "project-a", "repo", "deps")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := RegistryRef("ghcr.io/acme/app", "project-b", "repo", "deps")
	if first == second || first == "ghcr.io/acme/app" {
		t.Fatalf("registry refs are not isolated: %q %q", first, second)
	}
}

func TestLocalStoreSaveRestoreAndPurge(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "node_modules", "pkg"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "node_modules", "pkg", "index.js"), []byte("cached"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := LocalStore{Root: root}
	request := Request{Project: "project", Repository: "repo", Key: "lock-v1", Paths: []string{"node_modules"}, Workspace: workspace, MaxBytes: 1024}
	if err := store.Save(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(workspace, "node_modules")); err != nil {
		t.Fatal(err)
	}
	hit, err := store.Restore(ctx, request)
	if err != nil || !hit {
		t.Fatalf("Restore() = %v, %v", hit, err)
	}
	if data, err := os.ReadFile(filepath.Join(workspace, "node_modules", "pkg", "index.js")); err != nil || string(data) != "cached" {
		t.Fatalf("restored data = %q, %v", data, err)
	}
	if err := store.Purge(ctx, PurgeRequest{Project: "project"}); err != nil {
		t.Fatal(err)
	}
	hit, err = store.Restore(ctx, request)
	if err != nil || hit {
		t.Fatalf("Restore() after purge = %v, %v", hit, err)
	}
}

func TestLocalStoreEnforcesSizeAndSafePaths(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "large"), []byte("too large"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := LocalStore{Root: t.TempDir()}
	err := store.Save(context.Background(), Request{Project: "p", Repository: "r", Paths: []string{"large"}, Workspace: workspace, MaxBytes: 2})
	if err == nil {
		t.Fatal("Save() error = nil, want size error")
	}
	err = store.Save(context.Background(), Request{Project: "p", Repository: "r", Paths: []string{"../escape"}, Workspace: workspace})
	if err == nil {
		t.Fatalf("Save() path error = %v", err)
	}
}

func TestLocalStoreExpiresEntries(t *testing.T) {
	now := time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)
	store := LocalStore{Root: t.TempDir(), Now: func() time.Time { return now }}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "deps"), []byte("cached"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := Request{Project: "p", Repository: "r", Paths: []string{"deps"}, Workspace: workspace, TTL: time.Hour}
	if err := store.Save(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	hit, err := store.Restore(context.Background(), request)
	if err != nil || hit {
		t.Fatalf("Restore() expired = %v, %v", hit, err)
	}
}

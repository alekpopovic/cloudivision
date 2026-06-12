package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildKitBuilderRequiresImageRepository(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "buildctl")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake buildctl: %v", err)
	}
	t.Setenv("PATH", dir)

	_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{ContextDir: dir})
	if err == nil || !strings.Contains(err.Error(), "image repository is required") {
		t.Fatalf("Build() error = %v, want image repository validation", err)
	}
}

func TestBuildKitUnavailableGivesSetupGuidance(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{ImageRepository: "example/app"})
	if err == nil {
		t.Fatal("Build() error = nil, want BuildKit unavailable")
	}
	message := err.Error()
	if !strings.Contains(message, "BuildKit is not available") || !strings.Contains(message, "docker.sock") {
		t.Fatalf("Build() error = %q, want setup guidance without docker.sock dependency", message)
	}
}

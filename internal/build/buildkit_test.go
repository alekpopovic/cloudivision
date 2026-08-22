package build

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestBuildKitArgsIncludesOptionalSettingsInStableOrder(t *testing.T) {
	req := BuildRequest{
		ContextDir:      "/workspace/source/app",
		Dockerfile:      "docker/release.Dockerfile",
		ImageRepository: "registry.example/team/app",
		ImageTag:        "main-abc1234",
		Push:            true,
		BuildArgs:       map[string]string{"VERSION": "1.2.3", "GO_VERSION": "1.26"},
		Target:          "release",
		Platforms:       []string{"linux/amd64", "linux/arm64"},
		Labels:          map[string]string{"org.opencontainers.image.revision": "abc123", "app": "example"},
		Cache: CacheConfig{
			Enabled: true,
			Mode:    CacheModeRegistry,
			Ref:     "registry.example/team/app:buildcache",
		},
	}
	want := []string{
		"build",
		"--frontend", "dockerfile.v0",
		"--local", "context=/workspace/source/app",
		"--local", "dockerfile=/workspace/source/app/docker",
		"--opt", "filename=release.Dockerfile",
		"--opt", "build-arg:GO_VERSION=1.26",
		"--opt", "build-arg:VERSION=1.2.3",
		"--opt", "target=release",
		"--opt", "platform=linux/amd64,linux/arm64",
		"--opt", "label:app=example",
		"--opt", "label:org.opencontainers.image.revision=abc123",
		"--import-cache", "type=registry,ref=registry.example/team/app:buildcache",
		"--export-cache", "type=registry,ref=registry.example/team/app:buildcache,mode=max",
		"--output", "type=image,name=registry.example/team/app:main-abc1234,push=true",
		"--metadata-file", "/tmp/metadata.json",
	}

	got, err := buildKitArgs(req, "/tmp/metadata.json")
	if err != nil {
		t.Fatalf("buildKitArgs() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildKitArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildKitBuilderRequiresImageRepository(t *testing.T) {
	_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{ContextDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "image repository is required") {
		t.Fatalf("Build() error = %v, want image repository validation", err)
	}
	if got := FailureReason(err); got != ReasonBuildInputInvalid {
		t.Fatalf("FailureReason() = %q, want %q", got, ReasonBuildInputInvalid)
	}
}

func TestBuildKitBuilderRejectsMissingContextAndDockerfile(t *testing.T) {
	t.Run("context", func(t *testing.T) {
		_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{
			ContextDir: filepath.Join(t.TempDir(), "missing"), ImageRepository: "example/app",
		})
		if got := FailureReason(err); got != ReasonBuildContextMissing {
			t.Fatalf("FailureReason() = %q, want %q; error = %v", got, ReasonBuildContextMissing, err)
		}
	})
	t.Run("dockerfile", func(t *testing.T) {
		_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{
			ContextDir: t.TempDir(), Dockerfile: "docker/release.Dockerfile", ImageRepository: "example/app",
		})
		if got := FailureReason(err); got != ReasonDockerfileMissing {
			t.Fatalf("FailureReason() = %q, want %q; error = %v", got, ReasonDockerfileMissing, err)
		}
	})
	t.Run("escape", func(t *testing.T) {
		dir := buildContext(t)
		_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{
			ContextDir: dir, Dockerfile: "../Dockerfile", ImageRepository: "example/app",
		})
		if got := FailureReason(err); got != ReasonBuildInputInvalid {
			t.Fatalf("FailureReason() = %q, want %q; error = %v", got, ReasonBuildInputInvalid, err)
		}
	})
	t.Run("symlink escape", func(t *testing.T) {
		dir := t.TempDir()
		outside := filepath.Join(t.TempDir(), "Dockerfile")
		if err := os.WriteFile(outside, []byte("FROM scratch\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "Dockerfile")); err != nil {
			t.Fatal(err)
		}
		_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{
			ContextDir: dir, ImageRepository: "example/app",
		})
		if got := FailureReason(err); got != ReasonBuildInputInvalid {
			t.Fatalf("FailureReason() = %q, want %q; error = %v", got, ReasonBuildInputInvalid, err)
		}
	})
}

func TestBuildKitUnavailableGivesSetupGuidance(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := (BuildKitBuilder{}).Build(context.Background(), BuildRequest{
		ContextDir: buildContext(t), ImageRepository: "example/app",
	})
	if err == nil {
		t.Fatal("Build() error = nil, want BuildKit unavailable")
	}
	message := err.Error()
	if !strings.Contains(message, "BuildKit is not available") || !strings.Contains(message, "docker.sock") {
		t.Fatalf("Build() error = %q, want setup guidance without docker.sock dependency", message)
	}
	if got := FailureReason(err); got != ReasonBuildKitUnavailable {
		t.Fatalf("FailureReason() = %q, want %q", got, ReasonBuildKitUnavailable)
	}
}

func TestBuildKitBuilderCapturesDigest(t *testing.T) {
	binary := fakeBuildctl(t, `{"containerimage.digest":"`+testDigest+`"}`)
	result, err := (BuildKitBuilder{Binary: binary}).Build(context.Background(), BuildRequest{
		ContextDir: buildContext(t), ImageRepository: "example/app", ImageTag: "main", Push: true,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.Digest != testDigest || result.ImageRepository != "example/app" || result.Tag != "main" {
		t.Fatalf("Build() result = %#v", result)
	}
}

func TestBuildKitBuilderAllowsMissingDigestWhenPushDisabled(t *testing.T) {
	binary := fakeBuildctl(t, "")
	result, err := (BuildKitBuilder{Binary: binary}).Build(context.Background(), BuildRequest{
		ContextDir: buildContext(t), ImageRepository: "example/app", ImageTag: "dev", Push: false,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.Digest != "" || result.Tag != "dev" {
		t.Fatalf("Build() result = %#v", result)
	}
}

func TestBuildKitBuilderRequiresDigestForPush(t *testing.T) {
	binary := fakeBuildctl(t, "")
	_, err := (BuildKitBuilder{Binary: binary}).Build(context.Background(), BuildRequest{
		ContextDir: buildContext(t), ImageRepository: "example/app", Push: true,
	})
	if got := FailureReason(err); got != ReasonDigestCaptureFailed {
		t.Fatalf("FailureReason() = %q, want %q; error = %v", got, ReasonDigestCaptureFailed, err)
	}
}

func TestBuildKitBuilderReportsBuildFailure(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "buildctl")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\necho 'executor failed running build stage' >&2\nexit 9\n"), 0o755); err != nil {
		t.Fatalf("write fake buildctl: %v", err)
	}
	_, err := (BuildKitBuilder{Binary: binary}).Build(context.Background(), BuildRequest{
		ContextDir: buildContext(t), ImageRepository: "example/app", Push: true,
	})
	if got := FailureReason(err); got != ReasonImageBuildFailed {
		t.Fatalf("FailureReason() = %q, want %q", got, ReasonImageBuildFailed)
	}
	if !strings.Contains(err.Error(), "executor failed running build stage") {
		t.Fatalf("Build() error = %q, want bounded BuildKit diagnostic", err)
	}
}

func TestReadBuildKitDigest(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{name: "digest", content: `{"containerimage.digest":"` + testDigest + `"}`, want: testDigest},
		{name: "descriptor", content: `{"containerimage.descriptor":{"digest":"` + testDigest + `"}}`, want: testDigest},
		{name: "empty", content: ""},
		{name: "invalid json", content: "{", wantErr: true},
		{name: "invalid digest", content: `{"containerimage.digest":"sha256:short"}`, wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "metadata.json")
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := readBuildKitDigest(path)
			if (err != nil) != test.wantErr {
				t.Fatalf("readBuildKitDigest() error = %v, wantErr %t", err, test.wantErr)
			}
			if got != test.want {
				t.Fatalf("readBuildKitDigest() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildKitCacheRequiresReference(t *testing.T) {
	_, err := buildKitArgs(BuildRequest{Cache: CacheConfig{Enabled: true, Mode: CacheModeRegistry}}, "/tmp/metadata")
	if got := FailureReason(err); got != ReasonBuildInputInvalid {
		t.Fatalf("FailureReason() = %q, want %q", got, ReasonBuildInputInvalid)
	}
}

func TestResolveContextDirConfinesContextToSource(t *testing.T) {
	source := t.TempDir()
	inside := filepath.Join(source, "service")
	if err := os.Mkdir(inside, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveContextDir(source, "service")
	if err != nil || got != inside {
		t.Fatalf("ResolveContextDir() = %q, %v; want %q", got, err, inside)
	}
	if _, err := ResolveContextDir(source, "../outside"); FailureReason(err) != ReasonBuildInputInvalid {
		t.Fatalf("traversal error = %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(source, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveContextDir(source, "linked"); FailureReason(err) != ReasonBuildInputInvalid {
		t.Fatalf("symlink escape error = %v", err)
	}
}

func buildContext(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	return dir
}

func fakeBuildctl(t *testing.T, metadata string) string {
	t.Helper()
	dir := t.TempDir()
	binary := filepath.Join(dir, "buildctl")
	script := "#!/bin/sh\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--metadata-file\" ]; then\n" +
		"    shift\n" +
		"    printf '%s' '" + metadata + "' > \"$1\"\n" +
		"  fi\n" +
		"  shift\n" +
		"done\n"
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake buildctl: %v", err)
	}
	return binary
}

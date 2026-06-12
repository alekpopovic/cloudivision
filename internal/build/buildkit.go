package build

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type BuildKitBuilder struct {
	Timeout time.Duration
}

func (b BuildKitBuilder) Build(ctx context.Context, req BuildRequest) (*BuildResult, error) {
	binary, err := findBuildKitBinary()
	if err != nil {
		return nil, err
	}
	if req.ImageRepository == "" {
		return nil, fmt.Errorf("image repository is required")
	}
	if req.ContextDir == "" {
		req.ContextDir = "."
	}
	if req.Dockerfile == "" {
		req.Dockerfile = "Dockerfile"
	}
	tag := req.ImageTag
	if tag == "" {
		tag = "latest"
	}
	timeout := b.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute
	}
	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dockerfileDir := filepath.Dir(filepath.Join(req.ContextDir, req.Dockerfile))
	dockerfileName := filepath.Base(req.Dockerfile)
	output := fmt.Sprintf("type=image,name=%s:%s,push=%t", req.ImageRepository, tag, req.Push)
	args := []string{
		"build",
		"--frontend", "dockerfile.v0",
		"--local", "context=" + req.ContextDir,
		"--local", "dockerfile=" + dockerfileDir,
		"--opt", "filename=" + dockerfileName,
		"--output", output,
	}
	cmd := exec.CommandContext(buildCtx, binary, args...)
	cmd.Env = os.Environ()
	for key, value := range req.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	var stderr bytes.Buffer
	cmd.Stdout = &stderr
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("buildkit build failed: %s: %w", stderr.String(), err)
	}
	return &BuildResult{
		ImageRepository: req.ImageRepository,
		Tag:             tag,
	}, nil
}

func findBuildKitBinary() (string, error) {
	for _, name := range []string{"buildctl-daemonless.sh", "buildctl"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("BuildKit is not available: install buildctl or buildctl-daemonless.sh in the runner image; docker.sock and privileged Docker-in-Docker are not supported")
}

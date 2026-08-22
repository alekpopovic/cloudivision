package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ReasonBuildKitUnavailable  = "BuildKitUnavailable"
	ReasonBuildInputInvalid    = "ImageBuildInputInvalid"
	ReasonBuildContextMissing  = "ImageBuildContextMissing"
	ReasonDockerfileMissing    = "ImageBuildDockerfileMissing"
	ReasonImageBuildFailed     = "ImageBuildFailed"
	ReasonDigestCaptureFailed  = "ImageDigestCaptureFailed"
	defaultBuildTimeout        = 30 * time.Minute
	maxBuildKitDiagnosticBytes = 8 * 1024
)

var digestPattern = regexp.MustCompile(`^sha256:[a-fA-F0-9]{64}$`)

// Error is a sanitized builder failure with a stable BuildRun reason.
type Error struct {
	Reason  string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// FailureReason returns the stable reason carried by a builder error.
func FailureReason(err error) string {
	var buildErr *Error
	if errors.As(err, &buildErr) && buildErr.Reason != "" {
		return buildErr.Reason
	}
	return ReasonImageBuildFailed
}

type BuildKitBuilder struct {
	Timeout time.Duration
	Binary  string
}

// ResolveContextDir confines a PipelineTemplate context directory to the cloned
// source tree, including after symlink evaluation.
func ResolveContextDir(sourceDir, configured string) (string, error) {
	if configured == "" {
		configured = "."
	}
	if filepath.IsAbs(configured) {
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: "build context directory must be relative to the cloned repository"}
	}
	cleaned := filepath.Clean(configured)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: "build context directory must stay within the cloned repository"}
	}
	root, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: "resolve cloned repository directory", Err: err}
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: "resolve cloned repository directory", Err: err}
	}
	candidate := filepath.Join(root, cleaned)
	if _, err := os.Stat(candidate); err != nil {
		if os.IsNotExist(err) {
			return "", &Error{Reason: ReasonBuildContextMissing, Message: fmt.Sprintf("build context directory %q does not exist", configured)}
		}
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("inspect build context directory %q", configured), Err: err}
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("resolve build context directory %q", configured), Err: err}
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", &Error{Reason: ReasonBuildInputInvalid, Message: "build context directory symlink must stay within the cloned repository"}
	}
	return resolved, nil
}

func (b BuildKitBuilder) Build(ctx context.Context, req BuildRequest) (*BuildResult, error) {
	validated, err := validateBuildRequest(req)
	if err != nil {
		return nil, err
	}
	binary := b.Binary
	if binary == "" {
		binary, err = findBuildKitBinary()
		if err != nil {
			return nil, err
		}
	} else if _, err := exec.LookPath(binary); err != nil {
		return nil, &Error{
			Reason:  ReasonBuildKitUnavailable,
			Message: fmt.Sprintf("BuildKit executable %q is not available", binary),
			Err:     err,
		}
	}

	metadataFile, err := os.CreateTemp("", "cloudivision-buildkit-metadata-*.json")
	if err != nil {
		return nil, &Error{Reason: ReasonImageBuildFailed, Message: "create BuildKit metadata file", Err: err}
	}
	metadataPath := metadataFile.Name()
	if err := metadataFile.Close(); err != nil {
		_ = os.Remove(metadataPath)
		return nil, &Error{Reason: ReasonImageBuildFailed, Message: "close BuildKit metadata file", Err: err}
	}
	defer os.Remove(metadataPath)

	args, err := buildKitArgs(validated, metadataPath)
	if err != nil {
		return nil, err
	}
	timeout := b.Timeout
	if timeout <= 0 {
		timeout = defaultBuildTimeout
	}
	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(buildCtx, binary, args...)
	cmd.Env = append([]string{}, os.Environ()...)
	for _, key := range sortedKeys(validated.Env) {
		cmd.Env = append(cmd.Env, key+"="+validated.Env[key])
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if errors.Is(buildCtx.Err(), context.DeadlineExceeded) {
			return nil, &Error{
				Reason:  ReasonImageBuildFailed,
				Message: fmt.Sprintf("BuildKit build exceeded timeout %s", timeout),
				Err:     context.DeadlineExceeded,
			}
		}
		if errors.Is(buildCtx.Err(), context.Canceled) {
			return nil, &Error{Reason: ReasonImageBuildFailed, Message: "BuildKit build was cancelled", Err: context.Canceled}
		}
		diagnostic := boundedDiagnostic(output.String())
		message := "BuildKit build failed"
		if diagnostic != "" {
			message += ": " + diagnostic
		}
		return nil, &Error{Reason: ReasonImageBuildFailed, Message: message, Err: err}
	}

	digest, err := readBuildKitDigest(metadataPath)
	if err != nil {
		return nil, &Error{Reason: ReasonDigestCaptureFailed, Message: "capture image digest from BuildKit metadata", Err: err}
	}
	if validated.Push && digest == "" {
		return nil, &Error{
			Reason:  ReasonDigestCaptureFailed,
			Message: "BuildKit pushed the image but returned no sha256 manifest digest",
		}
	}
	return &BuildResult{
		ImageRepository: validated.ImageRepository,
		Tag:             validated.ImageTag,
		Digest:          digest,
	}, nil
}

func validateBuildRequest(req BuildRequest) (BuildRequest, error) {
	if strings.TrimSpace(req.ImageRepository) == "" {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: "image repository is required"}
	}
	if req.ContextDir == "" {
		req.ContextDir = "."
	}
	contextDir, err := filepath.Abs(filepath.Clean(req.ContextDir))
	if err != nil {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: "resolve build context", Err: err}
	}
	contextInfo, err := os.Stat(contextDir)
	if err != nil {
		if os.IsNotExist(err) {
			return req, &Error{Reason: ReasonBuildContextMissing, Message: fmt.Sprintf("build context directory %q does not exist", req.ContextDir)}
		}
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("inspect build context directory %q", req.ContextDir), Err: err}
	}
	if !contextInfo.IsDir() {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("build context %q is not a directory", req.ContextDir)}
	}
	resolvedContextDir, err := filepath.EvalSymlinks(contextDir)
	if err != nil {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("resolve build context directory %q", req.ContextDir), Err: err}
	}
	if req.Dockerfile == "" {
		req.Dockerfile = "Dockerfile"
	}
	if filepath.IsAbs(req.Dockerfile) {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: "Dockerfile path must be relative to the build context"}
	}
	dockerfilePath := filepath.Clean(filepath.Join(resolvedContextDir, req.Dockerfile))
	relativeDockerfile, err := filepath.Rel(resolvedContextDir, dockerfilePath)
	if err != nil || relativeDockerfile == ".." || strings.HasPrefix(relativeDockerfile, ".."+string(filepath.Separator)) {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: "Dockerfile path must stay within the build context"}
	}
	dockerfileInfo, err := os.Stat(dockerfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return req, &Error{Reason: ReasonDockerfileMissing, Message: fmt.Sprintf("Dockerfile %q does not exist in the build context", req.Dockerfile)}
		}
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("inspect Dockerfile %q", req.Dockerfile), Err: err}
	}
	if dockerfileInfo.IsDir() {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("Dockerfile %q is a directory", req.Dockerfile)}
	}
	resolvedDockerfile, err := filepath.EvalSymlinks(dockerfilePath)
	if err != nil {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("resolve Dockerfile %q", req.Dockerfile), Err: err}
	}
	relativeDockerfile, err = filepath.Rel(resolvedContextDir, resolvedDockerfile)
	if err != nil || relativeDockerfile == ".." || strings.HasPrefix(relativeDockerfile, ".."+string(filepath.Separator)) {
		return req, &Error{Reason: ReasonBuildInputInvalid, Message: "Dockerfile symlink must stay within the build context"}
	}
	if req.ImageTag == "" {
		req.ImageTag = "latest"
	}
	if err := validateCache(req.Cache); err != nil {
		return req, err
	}
	req.ContextDir = resolvedContextDir
	req.Dockerfile = relativeDockerfile
	return req, nil
}

func validateCache(cache CacheConfig) error {
	if !cache.Enabled {
		return nil
	}
	mode := cache.Mode
	if mode == "" {
		mode = CacheModeInline
	}
	switch mode {
	case CacheModeInline:
		return nil
	case CacheModeRegistry, CacheModeLocal:
		if strings.TrimSpace(cache.Ref) == "" {
			return &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("BuildKit %s cache requires a ref", mode)}
		}
		return nil
	default:
		return &Error{Reason: ReasonBuildInputInvalid, Message: fmt.Sprintf("unsupported BuildKit cache mode %q", cache.Mode)}
	}
}

func buildKitArgs(req BuildRequest, metadataPath string) ([]string, error) {
	if err := validateCache(req.Cache); err != nil {
		return nil, err
	}
	dockerfilePath := filepath.Join(req.ContextDir, req.Dockerfile)
	dockerfileDir := filepath.Dir(dockerfilePath)
	dockerfileName := filepath.Base(dockerfilePath)
	output := fmt.Sprintf("type=image,name=%s:%s,push=%t", req.ImageRepository, req.ImageTag, req.Push)
	args := []string{
		"build",
		"--frontend", "dockerfile.v0",
		"--local", "context=" + req.ContextDir,
		"--local", "dockerfile=" + dockerfileDir,
		"--opt", "filename=" + dockerfileName,
	}
	for _, key := range sortedKeys(req.BuildArgs) {
		args = append(args, "--opt", "build-arg:"+key+"="+req.BuildArgs[key])
	}
	if req.Target != "" {
		args = append(args, "--opt", "target="+req.Target)
	}
	if len(req.Platforms) > 0 {
		args = append(args, "--opt", "platform="+strings.Join(req.Platforms, ","))
	}
	for _, key := range sortedKeys(req.Labels) {
		args = append(args, "--opt", "label:"+key+"="+req.Labels[key])
	}
	if req.Cache.Enabled {
		mode := req.Cache.Mode
		if mode == "" {
			mode = CacheModeInline
		}
		switch mode {
		case CacheModeInline:
			args = append(args, "--export-cache", "type=inline")
		case CacheModeRegistry:
			args = append(args,
				"--import-cache", "type=registry,ref="+req.Cache.Ref,
				"--export-cache", "type=registry,ref="+req.Cache.Ref+",mode=max",
			)
		case CacheModeLocal:
			args = append(args,
				"--import-cache", "type=local,src="+req.Cache.Ref,
				"--export-cache", "type=local,dest="+req.Cache.Ref+",mode=max",
			)
		}
	}
	args = append(args, "--output", output, "--metadata-file", metadataPath)
	return args, nil
}

func findBuildKitBinary() (string, error) {
	for _, name := range []string{"buildctl-daemonless.sh", "buildctl"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", &Error{
		Reason:  ReasonBuildKitUnavailable,
		Message: "BuildKit is not available: install buildctl or buildctl-daemonless.sh in the runner image; docker.sock and privileged Docker-in-Docker are not supported",
	}
}

func readBuildKitDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return "", nil
	}
	metadata := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return "", fmt.Errorf("parse metadata JSON: %w", err)
	}
	for _, key := range []string{"containerimage.digest", "containerimage.descriptor"} {
		raw, ok := metadata[key]
		if !ok {
			continue
		}
		var digest string
		if key == "containerimage.descriptor" {
			var descriptor struct {
				Digest string `json:"digest"`
			}
			if err := json.Unmarshal(raw, &descriptor); err != nil {
				return "", fmt.Errorf("parse %s: %w", key, err)
			}
			digest = descriptor.Digest
		} else if err := json.Unmarshal(raw, &digest); err != nil {
			return "", fmt.Errorf("parse %s: %w", key, err)
		}
		if digest == "" {
			continue
		}
		if !digestPattern.MatchString(digest) {
			return "", fmt.Errorf("metadata contains invalid image digest %q", digest)
		}
		return strings.ToLower(digest), nil
	}
	return "", nil
}

func boundedDiagnostic(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= maxBuildKitDiagnosticBytes {
		return message
	}
	return message[:maxBuildKitDiagnosticBytes] + "... (truncated)"
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

package gitops

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"gopkg.in/yaml.v3"
)

var ErrDeploymentStatusUnavailable = errors.New("deployment status unavailable")

type Provider interface {
	UpdateImage(ctx context.Context, req UpdateImageRequest) (*UpdateImageResult, error)
	ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error)
}

type StatusReader interface {
	ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error)
}

type UpdateImageRequest struct {
	RepositoryURL string
	Branch        string
	Path          string
	Strategy      cicdv1alpha1.GitOpsStrategy
	ReleaseName   string
	Image         cicdv1alpha1.ImageRef
}

type UpdateImageResult struct {
	Commit string
}

type DeploymentStatusRequest struct {
	Provider        cicdv1alpha1.GitOpsProvider
	ApplicationName string
	Namespace       string
}

type DeploymentStatus struct {
	SyncStatus   string
	HealthStatus string
}

type Git interface {
	Clone(ctx context.Context, url, destination string) error
	CheckoutBranch(ctx context.Context, repositoryDir, branch string) error
	AddAll(ctx context.Context, repositoryDir string) error
	Commit(ctx context.Context, repositoryDir, message string) error
	Push(ctx context.Context, repositoryDir, branch string) error
	Head(ctx context.Context, repositoryDir string) (string, error)
}

type ExecGit struct{}

func (ExecGit) Clone(ctx context.Context, url, destination string) error {
	return runGit(ctx, "", "clone repository", "git", "clone", url, destination)
}

func (ExecGit) CheckoutBranch(ctx context.Context, repositoryDir, branch string) error {
	if branch == "" {
		return nil
	}
	return runGit(ctx, repositoryDir, "checkout branch", "git", "checkout", branch)
}

func (ExecGit) AddAll(ctx context.Context, repositoryDir string) error {
	return runGit(ctx, repositoryDir, "stage GitOps changes", "git", "add", ".")
}

func (ExecGit) Commit(ctx context.Context, repositoryDir, message string) error {
	return runGit(ctx, repositoryDir, "commit GitOps changes", "git",
		"-c", "user.name=cloudivision",
		"-c", "user.email=cloudivision@cloudivision.io",
		"commit", "-m", message)
}

func (ExecGit) Push(ctx context.Context, repositoryDir, branch string) error {
	args := []string{"push"}
	if branch != "" {
		args = append(args, "origin", branch)
	}
	return runGit(ctx, repositoryDir, "push GitOps changes", "git", args...)
}

func (ExecGit) Head(ctx context.Context, repositoryDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repositoryDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read GitOps HEAD: %s: %w", string(output), err)
	}
	return strings.TrimSpace(string(output)), nil
}

type GitRepositoryProvider struct {
	Git Git
}

func (p GitRepositoryProvider) UpdateImage(ctx context.Context, req UpdateImageRequest) (*UpdateImageResult, error) {
	if req.RepositoryURL == "" {
		return nil, errors.New("gitops repository URL is required")
	}
	if req.Image.Repository == "" {
		return nil, errors.New("release image repository is required")
	}
	strategy := req.Strategy
	if strategy == "" {
		strategy = cicdv1alpha1.GitOpsStrategyKustomizeImage
	}
	gitClient := p.Git
	if gitClient == nil {
		gitClient = ExecGit{}
	}

	workdir, err := os.MkdirTemp("", "cloudivision-gitops-*")
	if err != nil {
		return nil, fmt.Errorf("create GitOps workdir: %w", err)
	}
	defer os.RemoveAll(workdir)

	repoDir := filepath.Join(workdir, "repo")
	if err := gitClient.Clone(ctx, req.RepositoryURL, repoDir); err != nil {
		return nil, err
	}
	if err := gitClient.CheckoutBranch(ctx, repoDir, req.Branch); err != nil {
		return nil, err
	}
	if err := updateImageFiles(repoDir, req.Path, strategy, req.Image); err != nil {
		return nil, err
	}
	if err := gitClient.AddAll(ctx, repoDir); err != nil {
		return nil, err
	}
	message := fmt.Sprintf("cloudivision: release %s image %s", req.ReleaseName, imageString(req.Image))
	if err := gitClient.Commit(ctx, repoDir, message); err != nil {
		return nil, err
	}
	commit, err := gitClient.Head(ctx, repoDir)
	if err != nil {
		return nil, err
	}
	if err := gitClient.Push(ctx, repoDir, req.Branch); err != nil {
		return nil, err
	}
	return &UpdateImageResult{Commit: commit}, nil
}

func (p GitRepositoryProvider) ReadDeploymentStatus(context.Context, DeploymentStatusRequest) (*DeploymentStatus, error) {
	return nil, ErrDeploymentStatusUnavailable
}

func updateImageFiles(repoDir, targetPath string, strategy cicdv1alpha1.GitOpsStrategy, image cicdv1alpha1.ImageRef) error {
	switch strategy {
	case cicdv1alpha1.GitOpsStrategyHelmValues:
		return updateHelmValues(resolveGitOpsFile(repoDir, targetPath, "values.yaml"), image)
	case cicdv1alpha1.GitOpsStrategyKustomizeImage, "":
		return updateKustomization(resolveGitOpsFile(repoDir, targetPath, "kustomization.yaml"), image)
	case cicdv1alpha1.GitOpsStrategyRawYAML:
		return updateRawYAML(repoDir, targetPath, image)
	default:
		return fmt.Errorf("unsupported GitOps strategy %q", strategy)
	}
}

func resolveGitOpsFile(repoDir, targetPath, defaultFile string) string {
	if targetPath == "" {
		return filepath.Join(repoDir, defaultFile)
	}
	path := filepath.Join(repoDir, filepath.Clean(targetPath))
	if strings.HasSuffix(targetPath, ".yaml") || strings.HasSuffix(targetPath, ".yml") {
		return path
	}
	return filepath.Join(path, defaultFile)
}

func updateHelmValues(path string, image cicdv1alpha1.ImageRef) error {
	values, err := readYAMLMap(path)
	if err != nil {
		return err
	}
	imageValues, _ := values["image"].(map[string]any)
	if imageValues == nil {
		imageValues = map[string]any{}
		values["image"] = imageValues
	}
	imageValues["repository"] = image.Repository
	if image.Tag != "" {
		imageValues["tag"] = image.Tag
	}
	if image.Digest != "" {
		imageValues["digest"] = image.Digest
	}
	return writeYAML(path, values)
}

func updateKustomization(path string, image cicdv1alpha1.ImageRef) error {
	kustomization, err := readYAMLMap(path)
	if err != nil {
		return err
	}
	images, _ := kustomization["images"].([]any)
	updated := false
	for i := range images {
		item, ok := images[i].(map[string]any)
		if !ok {
			continue
		}
		name, _ := item["name"].(string)
		if name == image.Repository || name == "" {
			item["name"] = image.Repository
			item["newName"] = image.Repository
			if image.Tag != "" {
				item["newTag"] = image.Tag
			}
			if image.Digest != "" {
				item["digest"] = image.Digest
			}
			updated = true
			break
		}
	}
	if !updated {
		item := map[string]any{"name": image.Repository, "newName": image.Repository}
		if image.Tag != "" {
			item["newTag"] = image.Tag
		}
		if image.Digest != "" {
			item["digest"] = image.Digest
		}
		images = append(images, item)
	}
	kustomization["images"] = images
	return writeYAML(path, kustomization)
}

func updateRawYAML(repoDir, targetPath string, image cicdv1alpha1.ImageRef) error {
	path := filepath.Join(repoDir, filepath.Clean(targetPath))
	if targetPath == "" {
		path = repoDir
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("read raw YAML target: %w", err)
	}
	if !info.IsDir() {
		return updateWorkloadImage(path, image)
	}
	var changed bool
	err = filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !isYAML(path) {
			return err
		}
		updated, err := updateWorkloadImageIfPresent(path, image)
		if err != nil {
			return err
		}
		changed = changed || updated
		return nil
	})
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("no Deployment or StatefulSet image found under %s", path)
	}
	return nil
}

func updateWorkloadImage(path string, image cicdv1alpha1.ImageRef) error {
	updated, err := updateWorkloadImageIfPresent(path, image)
	if err != nil {
		return err
	}
	if !updated {
		return fmt.Errorf("no Deployment or StatefulSet image found in %s", path)
	}
	return nil
}

func updateWorkloadImageIfPresent(path string, image cicdv1alpha1.ImageRef) (bool, error) {
	manifest, err := readYAMLMap(path)
	if err != nil {
		return false, err
	}
	kind, _ := manifest["kind"].(string)
	if kind != "Deployment" && kind != "StatefulSet" {
		return false, nil
	}
	template, ok := nestedMap(manifest, "spec", "template", "spec")
	if !ok {
		return false, fmt.Errorf("workload %s has no pod template spec", path)
	}
	fullImage := imageString(image)
	updateContainerImages(template, "containers", fullImage)
	updateContainerImages(template, "initContainers", fullImage)
	return true, writeYAML(path, manifest)
}

func updateContainerImages(template map[string]any, field, image string) {
	containers, _ := template[field].([]any)
	for i := range containers {
		container, ok := containers[i].(map[string]any)
		if ok {
			container["image"] = image
		}
	}
}

func readYAMLMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read YAML %s: %w", path, err)
	}
	values := map[string]any{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parse YAML %s: %w", path, err)
	}
	return values, nil
}

func writeYAML(path string, values map[string]any) error {
	data, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal YAML %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write YAML %s: %w", path, err)
	}
	return nil
}

func nestedMap(root map[string]any, keys ...string) (map[string]any, bool) {
	current := root
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func imageString(image cicdv1alpha1.ImageRef) string {
	value := image.Repository
	if image.Tag != "" {
		value += ":" + image.Tag
	}
	if image.Digest != "" {
		value += "@" + image.Digest
	}
	return value
}

func isYAML(path string) bool {
	return strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
}

func runGit(ctx context.Context, dir, action string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s: %w", action, string(output), err)
	}
	return nil
}

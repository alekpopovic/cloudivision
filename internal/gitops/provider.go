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

type Operation string

const (
	OperationClone  Operation = "clone"
	OperationParse  Operation = "parse"
	OperationUpdate Operation = "update"
	OperationCommit Operation = "commit"
	OperationPush   Operation = "push"
)

// OperationError identifies the Git stage that failed without exposing credentials.
type OperationError struct {
	Operation Operation
	Err       error
}

func (e *OperationError) Error() string { return fmt.Sprintf("git %s failed: %v", e.Operation, e.Err) }
func (e *OperationError) Unwrap() error { return e.Err }

type Provider interface {
	UpdateImage(ctx context.Context, req UpdateImageRequest) (*UpdateImageResult, error)
	ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error)
}

type StatusReader interface {
	ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error)
}

type UpdateImageRequest struct {
	RepositoryURL        string
	Branch               string
	BaseBranch           string
	Path                 string
	Strategy             cicdv1alpha1.GitOpsStrategy
	ReleaseName          string
	Image                cicdv1alpha1.ImageRef
	ValuesFile           string
	ImageRepositoryField string
	ImageTagField        string
	ImageDigestField     string
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
	CreateBranch(ctx context.Context, repositoryDir, baseBranch, branch string) error
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

func (ExecGit) CreateBranch(ctx context.Context, repositoryDir, baseBranch, branch string) error {
	if baseBranch != "" {
		if err := runGit(ctx, repositoryDir, "checkout GitOps target branch", "git", "checkout", baseBranch); err != nil {
			return err
		}
	}
	return runGit(ctx, repositoryDir, "create GitOps promotion branch", "git", "checkout", "-B", branch)
}

func (ExecGit) AddAll(ctx context.Context, repositoryDir string) error {
	return runGit(ctx, repositoryDir, "stage GitOps changes", "git", "add", ".")
}

func (ExecGit) Commit(ctx context.Context, repositoryDir, message string) error {
	err := runGit(ctx, repositoryDir, "commit GitOps changes", "git",
		"-c", "user.name=cloudivision",
		"-c", "user.email=cloudivision@cloudivision.io",
		"commit", "-m", message)
	if err != nil && isNothingToCommit(err) {
		return nil
	}
	return err
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
		return nil, &OperationError{Operation: OperationClone, Err: err}
	}
	if req.BaseBranch != "" {
		err = gitClient.CreateBranch(ctx, repoDir, req.BaseBranch, req.Branch)
	} else {
		err = gitClient.CheckoutBranch(ctx, repoDir, req.Branch)
	}
	if err != nil {
		return nil, &OperationError{Operation: OperationClone, Err: err}
	}
	changed, err := updateImageFilesConfigured(repoDir, req.Path, strategy, req.Image, helmValuesConfig{
		ValuesFile: req.ValuesFile, RepositoryField: req.ImageRepositoryField,
		TagField: req.ImageTagField, DigestField: req.ImageDigestField,
	})
	if err != nil {
		operationError := &OperationError{}
		if errors.As(err, &operationError) {
			return nil, err
		}
		return nil, &OperationError{Operation: OperationUpdate, Err: err}
	}
	if !changed {
		commit, headErr := gitClient.Head(ctx, repoDir)
		if headErr != nil {
			return nil, &OperationError{Operation: OperationCommit, Err: headErr}
		}
		return &UpdateImageResult{Commit: commit}, nil
	}
	if err := gitClient.AddAll(ctx, repoDir); err != nil {
		return nil, &OperationError{Operation: OperationCommit, Err: err}
	}
	message := fmt.Sprintf("cloudivision: release %s image %s", req.ReleaseName, imageString(req.Image))
	if err := gitClient.Commit(ctx, repoDir, message); err != nil {
		return nil, &OperationError{Operation: OperationCommit, Err: err}
	}
	commit, err := gitClient.Head(ctx, repoDir)
	if err != nil {
		return nil, &OperationError{Operation: OperationCommit, Err: err}
	}
	if err := gitClient.Push(ctx, repoDir, req.Branch); err != nil {
		return nil, &OperationError{Operation: OperationPush, Err: err}
	}
	return &UpdateImageResult{Commit: commit}, nil
}

func (p GitRepositoryProvider) ReadDeploymentStatus(context.Context, DeploymentStatusRequest) (*DeploymentStatus, error) {
	return nil, ErrDeploymentStatusUnavailable
}

type helmValuesConfig struct {
	ValuesFile      string
	RepositoryField string
	TagField        string
	DigestField     string
}

func updateImageFiles(repoDir, targetPath string, strategy cicdv1alpha1.GitOpsStrategy, image cicdv1alpha1.ImageRef) error {
	_, err := updateImageFilesConfigured(repoDir, targetPath, strategy, image, helmValuesConfig{})
	return err
}

func updateImageFilesConfigured(repoDir, targetPath string, strategy cicdv1alpha1.GitOpsStrategy, image cicdv1alpha1.ImageRef, config helmValuesConfig) (bool, error) {
	switch strategy {
	case cicdv1alpha1.GitOpsStrategyHelmValues:
		path, err := helmValuesPath(repoDir, targetPath, config.ValuesFile)
		if err != nil {
			return false, &OperationError{Operation: OperationUpdate, Err: err}
		}
		return updateHelmValuesConfigured(path, image, config)
	case cicdv1alpha1.GitOpsStrategyKustomizeImage, "":
		path, err := safeGitOpsFile(repoDir, targetPath, "kustomization.yaml")
		if err != nil {
			return false, err
		}
		return true, updateKustomization(path, image)
	case cicdv1alpha1.GitOpsStrategyRawYAML:
		return true, updateRawYAML(repoDir, targetPath, image)
	default:
		return false, fmt.Errorf("unsupported GitOps strategy %q", strategy)
	}
}

func helmValuesPath(repoDir, targetPath, valuesFile string) (string, error) {
	if valuesFile == "" {
		valuesFile = "values.yaml"
	}
	if targetPath != "" && isYAML(targetPath) && valuesFile == "values.yaml" {
		return safeRepoPath(repoDir, targetPath)
	}
	return safeRepoPath(repoDir, filepath.Join(targetPath, valuesFile))
}

func safeGitOpsFile(repoDir, targetPath, defaultFile string) (string, error) {
	if targetPath == "" {
		return filepath.Join(repoDir, defaultFile), nil
	}
	path, err := safeRepoPath(repoDir, targetPath)
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(targetPath, ".yaml") || strings.HasSuffix(targetPath, ".yml") {
		return path, nil
	}
	return filepath.Join(path, defaultFile), nil
}

func safeRepoPath(repoDir, targetPath string) (string, error) {
	cleaned := filepath.Clean(targetPath)
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("gitops target path %q escapes repository root", targetPath)
	}
	path := filepath.Join(repoDir, cleaned)
	rel, err := filepath.Rel(repoDir, path)
	if err != nil {
		return "", fmt.Errorf("resolve gitops target path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("gitops target path %q escapes repository root", targetPath)
	}
	return path, nil
}

func updateHelmValues(path string, image cicdv1alpha1.ImageRef) error {
	_, err := updateHelmValuesConfigured(path, image, helmValuesConfig{})
	return err
}

func updateHelmValuesConfigured(path string, image cicdv1alpha1.ImageRef, config helmValuesConfig) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, &OperationError{Operation: OperationParse, Err: fmt.Errorf("read Helm values %s: %w", path, err)}
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return false, &OperationError{Operation: OperationParse, Err: fmt.Errorf("parse Helm values %s: %w", path, err)}
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return false, &OperationError{Operation: OperationParse, Err: fmt.Errorf("parse Helm values %s: document root must be a mapping", path)}
	}
	repositoryField := defaultString(config.RepositoryField, "image.repository")
	tagField := defaultString(config.TagField, "image.tag")
	digestField := defaultString(config.DigestField, "image.digest")
	changed, err := setYAMLField(document.Content[0], repositoryField, image.Repository)
	if err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: err}
	}
	fieldChanged, err := setOrDeleteYAMLField(document.Content[0], tagField, image.Tag)
	if err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: err}
	}
	changed = changed || fieldChanged
	if image.Digest != "" {
		fieldChanged, err = setYAMLField(document.Content[0], digestField, image.Digest)
	} else {
		fieldChanged, err = deleteYAMLField(document.Content[0], digestField)
	}
	if err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: err}
	}
	changed = changed || fieldChanged
	if !changed {
		return false, nil
	}
	var output strings.Builder
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("encode Helm values %s: %w", path, err)}
	}
	if err := encoder.Close(); err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("encode Helm values %s: %w", path, err)}
	}
	if err := os.WriteFile(path, []byte(output.String()), 0o644); err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("write Helm values %s: %w", path, err)}
	}
	return true, nil
}

func setOrDeleteYAMLField(root *yaml.Node, field, value string) (bool, error) {
	if value == "" {
		return deleteYAMLField(root, field)
	}
	return setYAMLField(root, field, value)
}

func setYAMLField(root *yaml.Node, field, value string) (bool, error) {
	parts, err := yamlFieldParts(field)
	if err != nil {
		return false, err
	}
	current := root
	for _, part := range parts[:len(parts)-1] {
		next := yamlMapValue(current, part)
		if next == nil {
			keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}
			next = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			current.Content = append(current.Content, keyNode, next)
		} else if next.Kind != yaml.MappingNode {
			return false, fmt.Errorf("Helm values field %q traverses non-map key %q", field, part)
		}
		current = next
	}
	leaf := parts[len(parts)-1]
	existing := yamlMapValue(current, leaf)
	if existing != nil {
		if existing.Kind != yaml.ScalarNode {
			return false, fmt.Errorf("Helm values field %q is not a scalar", field)
		}
		if existing.Value == value && existing.Tag == "!!str" {
			return false, nil
		}
		existing.Tag = "!!str"
		existing.Value = value
		return true, nil
	}
	current.Content = append(current.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: leaf},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
	return true, nil
}

func deleteYAMLField(root *yaml.Node, field string) (bool, error) {
	parts, err := yamlFieldParts(field)
	if err != nil {
		return false, err
	}
	current := root
	for _, part := range parts[:len(parts)-1] {
		next := yamlMapValue(current, part)
		if next == nil {
			return false, nil
		}
		if next.Kind != yaml.MappingNode {
			return false, fmt.Errorf("Helm values field %q traverses non-map key %q", field, part)
		}
		current = next
	}
	leaf := parts[len(parts)-1]
	for index := 0; index+1 < len(current.Content); index += 2 {
		if current.Content[index].Value == leaf {
			current.Content = append(current.Content[:index], current.Content[index+2:]...)
			return true, nil
		}
	}
	return false, nil
}

func yamlMapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func yamlFieldParts(field string) ([]string, error) {
	field = strings.TrimSpace(field)
	if field == "" {
		return nil, fmt.Errorf("Helm values field path is required")
	}
	parts := strings.Split(field, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf("Helm values field path %q contains an empty segment", field)
		}
	}
	return parts, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
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
			if image.Digest != "" {
				item["digest"] = image.Digest
				delete(item, "newTag")
			} else if image.Tag != "" {
				item["newTag"] = image.Tag
				delete(item, "digest")
			}
			updated = true
			break
		}
	}
	if !updated {
		item := map[string]any{"name": image.Repository, "newName": image.Repository}
		if image.Digest != "" {
			item["digest"] = image.Digest
		} else if image.Tag != "" {
			item["newTag"] = image.Tag
		}
		images = append(images, item)
	}
	kustomization["images"] = images
	return writeYAML(path, kustomization)
}

func updateRawYAML(repoDir, targetPath string, image cicdv1alpha1.ImageRef) error {
	path, err := safeRepoPath(repoDir, targetPath)
	if err != nil {
		return err
	}
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
	if image.Digest != "" {
		return value + "@" + image.Digest
	}
	if image.Tag != "" {
		value += ":" + image.Tag
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

func isNothingToCommit(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "nothing to commit") || strings.Contains(message, "no changes added to commit")
}

package gitops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"gopkg.in/yaml.v3"
)

var ErrDeploymentStatusUnavailable = errors.New("deployment status unavailable")
var ErrProviderUnavailable = errors.New("gitops provider unavailable")
var ErrDeploymentResourceMissing = errors.New("gitops deployment resource missing")

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

func (e *OperationError) Error() string {
	scope := "git"
	if e.Operation == OperationParse || e.Operation == OperationUpdate {
		scope = "gitops mutation"
	}
	return fmt.Sprintf("%s %s failed: %v", scope, e.Operation, e.Err)
}
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
	KustomizationFile    string
	KustomizeImageName   string
	RawYAMLFiles         []string
	WorkloadKind         string
	WorkloadName         string
	ContainerName        string
}

type UpdateImageResult struct {
	Commit string
}

type DeploymentStatusRequest struct {
	Provider        cicdv1alpha1.GitOpsProvider
	ApplicationName string
	Namespace       string
	ResourceKind    string
}

type DeploymentStatus struct {
	SyncStatus       string
	HealthStatus     string
	OperationPhase   string
	ObservedRevision string
	ObservedAt       string
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
	}, kustomizeConfig{KustomizationFile: req.KustomizationFile, ImageName: req.KustomizeImageName}, rawYAMLConfig{
		Files: req.RawYAMLFiles, WorkloadKind: req.WorkloadKind, WorkloadName: req.WorkloadName, ContainerName: req.ContainerName,
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

type kustomizeConfig struct {
	KustomizationFile string
	ImageName         string
}

type rawYAMLConfig struct {
	Files         []string
	WorkloadKind  string
	WorkloadName  string
	ContainerName string
}

func updateImageFiles(repoDir, targetPath string, strategy cicdv1alpha1.GitOpsStrategy, image cicdv1alpha1.ImageRef) error {
	_, err := updateImageFilesConfigured(repoDir, targetPath, strategy, image, helmValuesConfig{}, kustomizeConfig{}, rawYAMLConfig{})
	return err
}

func updateImageFilesConfigured(repoDir, targetPath string, strategy cicdv1alpha1.GitOpsStrategy, image cicdv1alpha1.ImageRef, helmConfig helmValuesConfig, kustomizeCfg kustomizeConfig, rawConfig rawYAMLConfig) (bool, error) {
	switch strategy {
	case cicdv1alpha1.GitOpsStrategyHelmValues:
		path, err := helmValuesPath(repoDir, targetPath, helmConfig.ValuesFile)
		if err != nil {
			return false, &OperationError{Operation: OperationUpdate, Err: err}
		}
		return updateHelmValuesConfigured(path, image, helmConfig)
	case cicdv1alpha1.GitOpsStrategyKustomizeImage, "":
		path, err := kustomizationPath(repoDir, targetPath, kustomizeCfg.KustomizationFile)
		if err != nil {
			return false, &OperationError{Operation: OperationUpdate, Err: err}
		}
		return updateKustomizationConfigured(path, image, kustomizeCfg.ImageName)
	case cicdv1alpha1.GitOpsStrategyRawYAML:
		return updateRawYAMLConfigured(repoDir, targetPath, image, rawConfig)
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
	if err := validateConfigFile(valuesFile, "valuesFile"); err != nil {
		return "", err
	}
	return safeRepoPath(repoDir, filepath.Join(targetPath, valuesFile))
}

func kustomizationPath(repoDir, targetPath, kustomizationFile string) (string, error) {
	if kustomizationFile == "" {
		kustomizationFile = "kustomization.yaml"
	}
	if targetPath != "" && isYAML(targetPath) && kustomizationFile == "kustomization.yaml" {
		return safeRepoPath(repoDir, targetPath)
	}
	if err := validateConfigFile(kustomizationFile, "kustomizationFile"); err != nil {
		return "", err
	}
	return safeRepoPath(repoDir, filepath.Join(targetPath, kustomizationFile))
}

func validateConfigFile(value, field string) error {
	cleaned := filepath.Clean(strings.TrimSpace(value))
	if cleaned == "." || cleaned == "" || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q must be a relative file path within gitOps.path", field, value)
	}
	return nil
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
	_, err := updateKustomizationConfigured(path, image, "")
	return err
}

func updateKustomizationConfigured(path string, image cicdv1alpha1.ImageRef, configuredImageName string) (bool, error) {
	kustomization, err := readYAMLMap(path)
	if err != nil {
		return false, &OperationError{Operation: OperationParse, Err: err}
	}
	images, imagesOK := kustomization["images"].([]any)
	if _, exists := kustomization["images"]; exists && !imagesOK {
		return false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("Kustomize images field in %s must be a list", path)}
	}
	imageName := configuredImageName
	if imageName == "" {
		imageName = image.Repository
	}
	updated := false
	changed := false
	for i := range images {
		item, ok := images[i].(map[string]any)
		if !ok {
			continue
		}
		name, _ := item["name"].(string)
		if name == imageName {
			changed = setMapString(item, "name", imageName) || changed
			changed = setMapString(item, "newName", image.Repository) || changed
			if image.Digest != "" {
				changed = setMapString(item, "digest", image.Digest) || changed
				changed = deleteMapKey(item, "newTag") || changed
			} else {
				changed = deleteMapKey(item, "digest") || changed
				if image.Tag != "" {
					changed = setMapString(item, "newTag", image.Tag) || changed
				} else {
					changed = deleteMapKey(item, "newTag") || changed
				}
			}
			updated = true
			break
		}
	}
	if !updated {
		if configuredImageName != "" {
			return false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("Kustomize image %q was not found in %s", configuredImageName, path)}
		}
		item := map[string]any{"name": imageName, "newName": image.Repository}
		if image.Digest != "" {
			item["digest"] = image.Digest
		} else if image.Tag != "" {
			item["newTag"] = image.Tag
		}
		images = append(images, item)
		changed = true
	}
	if !changed {
		return false, nil
	}
	kustomization["images"] = images
	if err := writeYAML(path, kustomization); err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: err}
	}
	return true, nil
}

func setMapString(values map[string]any, key, value string) bool {
	if existing, ok := values[key].(string); ok && existing == value {
		return false
	}
	values[key] = value
	return true
}

func deleteMapKey(values map[string]any, key string) bool {
	if _, exists := values[key]; !exists {
		return false
	}
	delete(values, key)
	return true
}

func updateRawYAML(repoDir, targetPath string, image cicdv1alpha1.ImageRef) error {
	_, err := updateRawYAMLConfigured(repoDir, targetPath, image, rawYAMLConfig{})
	return err
}

func updateRawYAMLConfigured(repoDir, targetPath string, image cicdv1alpha1.ImageRef, config rawYAMLConfig) (bool, error) {
	paths, err := rawYAMLPaths(repoDir, targetPath, config.Files)
	if err != nil {
		return false, &OperationError{Operation: OperationUpdate, Err: err}
	}
	matched := false
	changed := false
	for _, path := range paths {
		fileMatched, fileChanged, updateErr := updateRawYAMLFile(path, image, config)
		if updateErr != nil {
			return false, updateErr
		}
		matched = matched || fileMatched
		changed = changed || fileChanged
	}
	if !matched {
		return false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("no matching raw YAML workload/container found under %q", targetPath)}
	}
	return changed, nil
}

func rawYAMLPaths(repoDir, targetPath string, files []string) ([]string, error) {
	if len(files) > 0 {
		paths := make([]string, 0, len(files))
		for _, file := range files {
			if err := validateConfigFile(file, "rawYaml.files entry"); err != nil {
				return nil, err
			}
			path, err := safeRepoPath(repoDir, filepath.Join(targetPath, file))
			if err != nil {
				return nil, err
			}
			paths = append(paths, path)
		}
		return paths, nil
	}
	path := repoDir
	if targetPath != "" {
		var err error
		path, err = safeRepoPath(repoDir, targetPath)
		if err != nil {
			return nil, err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("read raw YAML target: %w", err)
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	paths := []string{}
	err = filepath.WalkDir(path, func(candidate string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && isYAML(candidate) {
			paths = append(paths, candidate)
		}
		return nil
	})
	return paths, err
}

func updateRawYAMLFile(path string, image cicdv1alpha1.ImageRef, config rawYAMLConfig) (bool, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, false, &OperationError{Operation: OperationParse, Err: fmt.Errorf("read raw YAML %s: %w", path, err)}
	}
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	documents := []*yaml.Node{}
	matched := false
	changed := false
	for {
		document := &yaml.Node{}
		if err := decoder.Decode(document); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return false, false, &OperationError{Operation: OperationParse, Err: fmt.Errorf("parse raw YAML %s: %w", path, err)}
		}
		documents = append(documents, document)
		if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
			continue
		}
		documentMatched, documentChanged, updateErr := updateRawWorkloadNode(document.Content[0], image, config)
		if updateErr != nil {
			return false, false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("update raw YAML %s: %w", path, updateErr)}
		}
		matched = matched || documentMatched
		changed = changed || documentChanged
	}
	if !changed {
		return matched, false, nil
	}
	var output strings.Builder
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	for _, document := range documents {
		if err := encoder.Encode(document); err != nil {
			return false, false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("encode raw YAML %s: %w", path, err)}
		}
	}
	if err := encoder.Close(); err != nil {
		return false, false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("encode raw YAML %s: %w", path, err)}
	}
	if err := os.WriteFile(path, []byte(output.String()), 0o644); err != nil {
		return false, false, &OperationError{Operation: OperationUpdate, Err: fmt.Errorf("write raw YAML %s: %w", path, err)}
	}
	return matched, true, nil
}

func updateRawWorkloadNode(root *yaml.Node, image cicdv1alpha1.ImageRef, config rawYAMLConfig) (bool, bool, error) {
	kindNode := yamlMapValue(root, "kind")
	if kindNode == nil || !supportedWorkloadKind(kindNode.Value) || config.WorkloadKind != "" && kindNode.Value != config.WorkloadKind {
		return false, false, nil
	}
	metadata := yamlMapValue(root, "metadata")
	nameNode := yamlMapValue(metadata, "name")
	if config.WorkloadName != "" && (nameNode == nil || nameNode.Value != config.WorkloadName) {
		return false, false, nil
	}
	var podSpec *yaml.Node
	if kindNode.Value == "CronJob" {
		podSpec = yamlNodeAt(root, "spec", "jobTemplate", "spec", "template", "spec")
	} else {
		podSpec = yamlNodeAt(root, "spec", "template", "spec")
	}
	if podSpec == nil || podSpec.Kind != yaml.MappingNode {
		return false, false, fmt.Errorf("%s %q has no pod template spec", kindNode.Value, nodeValue(nameNode))
	}
	candidates := rawContainerCandidates(podSpec)
	selected := []rawContainer{}
	if config.ContainerName != "" {
		for _, candidate := range candidates {
			if candidate.Name == config.ContainerName {
				selected = append(selected, candidate)
			}
		}
	} else {
		for _, candidate := range candidates {
			if imageRepository(candidate.Image.Value) == image.Repository {
				selected = append(selected, candidate)
			}
		}
		if len(selected) == 0 && len(candidates) == 1 {
			selected = candidates
		}
	}
	if len(selected) == 0 {
		return false, false, nil
	}
	desired := imageString(image)
	changed := false
	for _, candidate := range selected {
		if candidate.Image.Value != desired {
			candidate.Image.Value = desired
			candidate.Image.Tag = "!!str"
			changed = true
		}
	}
	return true, changed, nil
}

type rawContainer struct {
	Name  string
	Image *yaml.Node
}

func rawContainerCandidates(podSpec *yaml.Node) []rawContainer {
	result := []rawContainer{}
	for _, field := range []string{"containers", "initContainers"} {
		sequence := yamlMapValue(podSpec, field)
		if sequence == nil || sequence.Kind != yaml.SequenceNode {
			continue
		}
		for _, item := range sequence.Content {
			if item.Kind != yaml.MappingNode {
				continue
			}
			name := yamlMapValue(item, "name")
			image := yamlMapValue(item, "image")
			if name != nil && image != nil && image.Kind == yaml.ScalarNode {
				result = append(result, rawContainer{Name: name.Value, Image: image})
			}
		}
	}
	return result
}

func yamlNodeAt(root *yaml.Node, keys ...string) *yaml.Node {
	current := root
	for _, key := range keys {
		current = yamlMapValue(current, key)
		if current == nil {
			return nil
		}
	}
	return current
}

func nodeValue(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	return node.Value
}

func supportedWorkloadKind(kind string) bool {
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "CronJob":
		return true
	default:
		return false
	}
}

func imageRepository(value string) string {
	if index := strings.Index(value, "@"); index >= 0 {
		value = value[:index]
	}
	lastSlash := strings.LastIndex(value, "/")
	if index := strings.LastIndex(value, ":"); index > lastSlash {
		value = value[:index]
	}
	return value
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

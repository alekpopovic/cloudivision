package gitops

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"gopkg.in/yaml.v3"
)

func TestGitRepositoryProviderUpdatesKustomizeImage(t *testing.T) {
	ctx := context.Background()
	remote := createRemoteGitOpsRepo(t, map[string]string{
		"kustomization.yaml": "resources:\n- deployment.yaml\nimages:\n- name: ghcr.io/cloudivision/example\n  newName: ghcr.io/cloudivision/example\n  newTag: old\n",
	})

	result, err := (GitRepositoryProvider{}).UpdateImage(ctx, UpdateImageRequest{
		RepositoryURL: remote,
		Branch:        "main",
		Strategy:      cicdv1alpha1.GitOpsStrategyKustomizeImage,
		ReleaseName:   "sample-release",
		Image: cicdv1alpha1.ImageRef{
			Repository: "ghcr.io/cloudivision/example",
			Tag:        "new",
		},
	})
	if err != nil {
		t.Fatalf("UpdateImage() error = %v", err)
	}
	if result.Commit == "" {
		t.Fatal("commit is empty")
	}

	clone := filepath.Join(t.TempDir(), "clone")
	runGitTest(t, "", "git", "clone", remote, clone)
	data, err := os.ReadFile(filepath.Join(clone, "kustomization.yaml"))
	if err != nil {
		t.Fatalf("read kustomization: %v", err)
	}
	if !strings.Contains(string(data), "newTag: new") {
		t.Fatalf("kustomization.yaml =\n%s", string(data))
	}
}

func TestGitRepositoryProviderTreatsNoOpImageUpdateAsSuccess(t *testing.T) {
	ctx := context.Background()
	remote := createRemoteGitOpsRepo(t, map[string]string{
		"kustomization.yaml": "resources:\n- deployment.yaml\nimages:\n- name: ghcr.io/cloudivision/example\n  newName: ghcr.io/cloudivision/example\n  newTag: current\n",
	})

	result, err := (GitRepositoryProvider{}).UpdateImage(ctx, UpdateImageRequest{
		RepositoryURL: remote,
		Branch:        "main",
		Strategy:      cicdv1alpha1.GitOpsStrategyKustomizeImage,
		ReleaseName:   "sample-release",
		Image: cicdv1alpha1.ImageRef{
			Repository: "ghcr.io/cloudivision/example",
			Tag:        "current",
		},
	})
	if err != nil {
		t.Fatalf("UpdateImage() error = %v", err)
	}
	if result.Commit == "" {
		t.Fatal("commit is empty")
	}
}

func TestGitRepositoryProviderCreatesPromotionBranchFromTarget(t *testing.T) {
	ctx := context.Background()
	remote := createRemoteGitOpsRepo(t, map[string]string{
		"kustomization.yaml": "images:\n- name: ghcr.io/cloudivision/example\n  newTag: old\n",
	})
	result, err := (GitRepositoryProvider{}).UpdateImage(ctx, UpdateImageRequest{
		RepositoryURL: remote,
		Branch:        "cloudivision/release-1",
		BaseBranch:    "main",
		Strategy:      cicdv1alpha1.GitOpsStrategyKustomizeImage,
		ReleaseName:   "release-1",
		Image:         cicdv1alpha1.ImageRef{Repository: "ghcr.io/cloudivision/example", Tag: "v2"},
	})
	if err != nil {
		t.Fatalf("UpdateImage() error = %v", err)
	}
	output := runGitTestOutput(t, "", "git", "--git-dir", remote, "rev-parse", "refs/heads/cloudivision/release-1")
	if strings.TrimSpace(output) != result.Commit {
		t.Fatalf("promotion branch commit = %q, want %q", strings.TrimSpace(output), result.Commit)
	}
}

func TestUpdateHelmValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(path, []byte("image:\n  repository: old\n  tag: old\n"), 0o644); err != nil {
		t.Fatalf("write values: %v", err)
	}

	err := updateImageFiles(dir, "values.yaml", cicdv1alpha1.GitOpsStrategyHelmValues, cicdv1alpha1.ImageRef{
		Repository: "ghcr.io/cloudivision/example",
		Tag:        "v1",
		Digest:     "sha256:123",
	})
	if err != nil {
		t.Fatalf("updateImageFiles() error = %v", err)
	}
	values := readMapForTest(t, path)
	image := values["image"].(map[string]any)
	if image["repository"] != "ghcr.io/cloudivision/example" || image["digest"] != "sha256:123" {
		t.Fatalf("image values = %#v", image)
	}
	if image["tag"] != "v1" {
		t.Fatalf("image tag = %#v, want v1", image["tag"])
	}
}

func TestUpdateHelmValuesUsesNestedConfiguredFieldsAndPreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "production.yaml")
	content := "# production values\nworkloads:\n  api:\n    container:\n      repository: old # keep this comment\n      tag: old\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := updateHelmValuesConfigured(path, cicdv1alpha1.ImageRef{Repository: "ghcr.io/acme/api", Tag: "v2", Digest: "sha256:abc"}, helmValuesConfig{
		RepositoryField: "workloads.api.container.repository",
		TagField:        "workloads.api.container.tag",
		DigestField:     "workloads.api.container.digest",
	})
	if err != nil || !changed {
		t.Fatalf("updateHelmValuesConfigured() = %v, %v", changed, err)
	}
	updated := string(mustRead(t, path))
	if !strings.Contains(updated, "# production values") || !strings.Contains(updated, "# keep this comment") || !strings.Contains(updated, "digest: sha256:abc") {
		t.Fatalf("production.yaml =\n%s", updated)
	}
}

func TestUpdateHelmValuesReportsMissingAndInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	for _, test := range []struct {
		name string
		path string
	}{
		{name: "missing", path: filepath.Join(dir, "missing.yaml")},
		{name: "invalid", path: filepath.Join(dir, "invalid.yaml")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "invalid" {
				if err := os.WriteFile(test.path, []byte("image: [unterminated"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			_, err := updateHelmValuesConfigured(test.path, cicdv1alpha1.ImageRef{Repository: "example/app", Tag: "v1"}, helmValuesConfig{})
			operationError := &OperationError{}
			if !errors.As(err, &operationError) || operationError.Operation != OperationParse {
				t.Fatalf("error = %v, want parse OperationError", err)
			}
		})
	}
}

func TestGitRepositoryProviderHelmValuesIsIdempotent(t *testing.T) {
	remote := createRemoteGitOpsRepo(t, map[string]string{
		"apps/api/production.yaml": "# keep\nimage:\n  repository: old\n  tag: old\n",
	})
	request := UpdateImageRequest{
		RepositoryURL: remote, Branch: "main", Path: "apps/api", ValuesFile: "production.yaml",
		Strategy: cicdv1alpha1.GitOpsStrategyHelmValues, ReleaseName: "release-1",
		Image: cicdv1alpha1.ImageRef{Repository: "ghcr.io/acme/api", Tag: "v2", Digest: "sha256:abc"},
	}
	provider := GitRepositoryProvider{}
	first, err := provider.UpdateImage(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := provider.UpdateImage(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Commit == "" || first.Commit != second.Commit {
		t.Fatalf("commits = %q, %q", first.Commit, second.Commit)
	}
	count := strings.TrimSpace(runGitTestOutput(t, "", "git", "--git-dir", remote, "rev-list", "--count", "main"))
	if count != "2" {
		t.Fatalf("commit count = %s, want initial plus one release commit", count)
	}
}

func TestGitRepositoryProviderReportsPushFailure(t *testing.T) {
	remote := createRemoteGitOpsRepo(t, map[string]string{"values.yaml": "image:\n  repository: old\n  tag: old\n"})
	hook := filepath.Join(remote, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := (GitRepositoryProvider{}).UpdateImage(context.Background(), UpdateImageRequest{
		RepositoryURL: remote, Branch: "main", Strategy: cicdv1alpha1.GitOpsStrategyHelmValues,
		ReleaseName: "release-1", Image: cicdv1alpha1.ImageRef{Repository: "example/app", Tag: "v2"},
	})
	operationError := &OperationError{}
	if !errors.As(err, &operationError) || operationError.Operation != OperationPush {
		t.Fatalf("error = %v, want push OperationError", err)
	}
}

func TestUpdateKustomizationPrefersDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kustomization.yaml")
	if err := os.WriteFile(path, []byte("images:\n- name: ghcr.io/cloudivision/example\n  newTag: old\n"), 0o644); err != nil {
		t.Fatalf("write kustomization: %v", err)
	}

	err := updateImageFiles(dir, "kustomization.yaml", cicdv1alpha1.GitOpsStrategyKustomizeImage, cicdv1alpha1.ImageRef{
		Repository: "ghcr.io/cloudivision/example",
		Tag:        "mutable",
		Digest:     "sha256:123",
	})
	if err != nil {
		t.Fatalf("updateImageFiles() error = %v", err)
	}
	content := string(mustRead(t, path))
	if !strings.Contains(content, "digest: sha256:123") || strings.Contains(content, "newTag:") {
		t.Fatalf("kustomization.yaml =\n%s", content)
	}
}

func TestUpdateImageFilesRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := updateImageFiles(dir, "../outside.yaml", cicdv1alpha1.GitOpsStrategyHelmValues, cicdv1alpha1.ImageRef{Repository: "example"}); err == nil {
		t.Fatal("updateImageFiles() error = nil, want path traversal rejection")
	}
	if err := updateImageFiles(dir, "/tmp/outside.yaml", cicdv1alpha1.GitOpsStrategyKustomizeImage, cicdv1alpha1.ImageRef{Repository: "example"}); err == nil {
		t.Fatal("updateImageFiles() error = nil, want absolute path rejection")
	}
}

func TestUpdateRawYAMLWorkloadImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployment.yaml")
	if err := os.WriteFile(path, []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample
spec:
  template:
    spec:
      containers:
      - name: app
        image: old:tag
`), 0o644); err != nil {
		t.Fatalf("write deployment: %v", err)
	}

	err := updateImageFiles(dir, "deployment.yaml", cicdv1alpha1.GitOpsStrategyRawYAML, cicdv1alpha1.ImageRef{
		Repository: "ghcr.io/cloudivision/example",
		Tag:        "v2",
	})
	if err != nil {
		t.Fatalf("updateImageFiles() error = %v", err)
	}
	if !strings.Contains(string(mustRead(t, path)), "image: ghcr.io/cloudivision/example:v2") {
		t.Fatalf("deployment.yaml =\n%s", string(mustRead(t, path)))
	}
}

func TestUpdateRawYAMLWorkloadPrefersDigest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deployment.yaml")
	if err := os.WriteFile(path, []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample
spec:
  template:
    spec:
      containers:
      - name: app
        image: old:tag
`), 0o644); err != nil {
		t.Fatalf("write deployment: %v", err)
	}

	err := updateImageFiles(dir, "deployment.yaml", cicdv1alpha1.GitOpsStrategyRawYAML, cicdv1alpha1.ImageRef{
		Repository: "ghcr.io/cloudivision/example",
		Tag:        "mutable",
		Digest:     "sha256:123",
	})
	if err != nil {
		t.Fatalf("updateImageFiles() error = %v", err)
	}
	if !strings.Contains(string(mustRead(t, path)), "image: ghcr.io/cloudivision/example@sha256:123") {
		t.Fatalf("deployment.yaml =\n%s", string(mustRead(t, path)))
	}
}

func createRemoteGitOpsRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	work := filepath.Join(root, "work")
	runGitTest(t, "", "git", "init", "--bare", remote)
	runGitTest(t, "", "git", "init", work)
	runGitTest(t, work, "git", "checkout", "-b", "main")
	for name, content := range files {
		path := filepath.Join(work, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create dir for %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	runGitTest(t, work, "git", "add", ".")
	runGitTest(t, work, "git", "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	runGitTest(t, work, "git", "remote", "add", "origin", remote)
	runGitTest(t, work, "git", "push", "origin", "main")
	runGitTest(t, "", "git", "--git-dir", remote, "symbolic-ref", "HEAD", "refs/heads/main")
	return remote
}

func runGitTest(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %s: %v", name, strings.Join(args, " "), string(output), err)
	}
}

func runGitTestOutput(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %s: %v", name, strings.Join(args, " "), string(output), err)
	}
	return string(output)
}

func readMapForTest(t *testing.T, path string) map[string]any {
	t.Helper()
	values := map[string]any{}
	if err := yaml.Unmarshal(mustRead(t, path), &values); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return values
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

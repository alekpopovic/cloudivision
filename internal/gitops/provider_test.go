package gitops

import (
	"context"
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
	if image["repository"] != "ghcr.io/cloudivision/example" || image["tag"] != "v1" || image["digest"] != "sha256:123" {
		t.Fatalf("image values = %#v", image)
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

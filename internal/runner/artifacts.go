package runner

import (
	"context"
	"crypto/sha256"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/artifacts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const maxArtifactBytes int64 = 100 << 20

var artifactNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

func (r Runner) collectArtifacts(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, template *cicdv1alpha1.PipelineTemplate, sourceDir string) ([]cicdv1alpha1.BuildRunArtifactStatus, error) {
	root, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolve source workspace: %w", err)
	}
	statuses := []cicdv1alpha1.BuildRunArtifactStatus{}
	usedNames := map[string]int{}
	for _, step := range template.Spec.Steps {
		if step.Artifacts == nil {
			continue
		}
		for _, pattern := range step.Artifacts.Paths {
			if err := validateArtifactPattern(pattern); err != nil {
				return nil, fmt.Errorf("step %q artifact path %q: %w", step.Name, pattern, err)
			}
			matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
			if err != nil {
				return nil, fmt.Errorf("step %q artifact glob %q: %w", step.Name, pattern, err)
			}
			files, err := expandArtifactFiles(root, matches)
			if err != nil {
				return nil, err
			}
			if len(files) == 0 && !step.Artifacts.Optional {
				return nil, fmt.Errorf("required artifact path %q did not match a regular file", pattern)
			}
			for _, file := range files {
				data, err := os.ReadFile(file)
				if err != nil {
					return nil, fmt.Errorf("read artifact %q: %w", file, err)
				}
				if int64(len(data)) > maxArtifactBytes {
					return nil, fmt.Errorf("artifact %q exceeds the %d byte limit", file, maxArtifactBytes)
				}
				relative, _ := filepath.Rel(root, file)
				name := uniqueArtifactName(step.Name+"-"+filepath.Base(file), usedNames)
				digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
				mediaType := mime.TypeByExtension(filepath.Ext(file))
				if mediaType == "" {
					mediaType = "application/octet-stream"
				}
				ref, err := r.ArtifactStore.Put(ctx, artifacts.PutArtifactRequest{Namespace: buildRun.Namespace, BuildRun: buildRun.Name, Name: name, Path: filepath.ToSlash(relative), Type: mediaType, Digest: digest, Data: data, RetentionDays: step.Artifacts.RetentionDays})
				if err != nil {
					return nil, fmt.Errorf("upload artifact %q: %w", relative, err)
				}
				createdAt := metav1.NewTime(ref.CreatedAt)
				statuses = append(statuses, cicdv1alpha1.BuildRunArtifactStatus{Name: ref.Name, Path: ref.Path, Type: ref.Type, Size: ref.Size, Digest: ref.Digest, Ref: ref.Ref, CreatedAt: &createdAt})
			}
		}
	}
	return statuses, nil
}

func validateArtifactPattern(pattern string) error {
	if pattern == "" || filepath.IsAbs(pattern) {
		return fmt.Errorf("must be a non-empty relative path")
	}
	clean := filepath.Clean(pattern)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("must remain inside the source workspace")
	}
	for _, component := range strings.FieldsFunc(strings.ToLower(filepath.ToSlash(clean)), func(r rune) bool { return r == '/' || r == '\\' }) {
		if component == ".git" || component == ".env" || strings.Contains(component, "secret") || strings.Contains(component, "password") || strings.Contains(component, "private-key") || strings.Contains(component, "credentials") {
			return fmt.Errorf("matches a protected secret-like path component")
		}
	}
	return nil
}

func expandArtifactFiles(root string, matches []string) ([]string, error) {
	files := []string{}
	for _, match := range matches {
		resolved, err := filepath.EvalSymlinks(match)
		if err != nil {
			return nil, fmt.Errorf("resolve artifact path %q: %w", match, err)
		}
		if resolved != root && !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			return nil, fmt.Errorf("artifact path %q escapes the source workspace", match)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			relative, _ := filepath.Rel(root, resolved)
			if err := validateArtifactPattern(relative); err != nil {
				return nil, fmt.Errorf("artifact file %q is protected: %w", relative, err)
			}
			files = append(files, resolved)
			continue
		}
		if info.IsDir() {
			err := filepath.WalkDir(resolved, func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.Type().IsRegular() {
					relative, _ := filepath.Rel(root, path)
					if err := validateArtifactPattern(relative); err != nil {
						return fmt.Errorf("artifact file %q is protected: %w", relative, err)
					}
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}
	sort.Strings(files)
	return files, nil
}

func uniqueArtifactName(value string, used map[string]int) string {
	name := strings.Trim(artifactNameSanitizer.ReplaceAllString(value, "-"), "-")
	if name == "" {
		name = "artifact"
	}
	used[name]++
	if used[name] > 1 {
		name = fmt.Sprintf("%s-%d", name, used[name])
	}
	return name
}

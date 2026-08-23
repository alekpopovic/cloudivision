package artifacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

type LocalStore struct{ Root string }

func (s LocalStore) paths(namespace, buildRun, name string) (string, string, error) {
	if !safeName.MatchString(namespace) || !safeName.MatchString(buildRun) || !safeName.MatchString(name) {
		return "", "", errors.New("namespace, BuildRun, and artifact name must be safe path components")
	}
	if s.Root == "" {
		return "", "", errors.New("local artifact root is required")
	}
	base := filepath.Join(s.Root, namespace, buildRun)
	return filepath.Join(base, name), filepath.Join(base, name+".metadata.json"), nil
}

func (s LocalStore) Put(_ context.Context, req PutArtifactRequest) (*ArtifactRef, error) {
	path, metadataPath, err := s.paths(req.Namespace, req.BuildRun, req.Name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create artifact directory: %w", err)
	}
	if err := os.WriteFile(path, req.Data, 0o600); err != nil {
		return nil, fmt.Errorf("write artifact: %w", err)
	}
	ref := ArtifactRef{Namespace: req.Namespace, BuildRun: req.BuildRun, Name: req.Name, Path: req.Path, Type: req.Type, Size: int64(len(req.Data)), Digest: req.Digest, Ref: path, CreatedAt: time.Now().UTC()}
	metadata, err := json.Marshal(ref)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(metadataPath, metadata, 0o600); err != nil {
		return nil, fmt.Errorf("write artifact metadata: %w", err)
	}
	return &ref, nil
}

func (s LocalStore) Get(_ context.Context, ref ArtifactRef) (*Artifact, error) {
	path, _, err := s.paths(ref.Namespace, ref.BuildRun, ref.Name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read artifact: %w", err)
	}
	return &Artifact{Ref: ref, Data: data}, nil
}

func (s LocalStore) List(_ context.Context, req ListArtifactsRequest) ([]ArtifactRef, error) {
	base := filepath.Join(s.Root, req.Namespace, req.BuildRun)
	entries, err := filepath.Glob(filepath.Join(base, "*.metadata.json"))
	if err != nil {
		return nil, err
	}
	refs := make([]ArtifactRef, 0, len(entries))
	for _, entry := range entries {
		data, readErr := os.ReadFile(entry)
		if readErr != nil {
			return nil, readErr
		}
		var ref ArtifactRef
		if err := json.Unmarshal(data, &ref); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs, nil
}

func (s LocalStore) Delete(_ context.Context, ref ArtifactRef) error {
	path, metadataPath, err := s.paths(ref.Namespace, ref.BuildRun, ref.Name)
	if err != nil {
		return err
	}
	for _, target := range []string{path, metadataPath} {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrUnsupported = errors.New("dependency cache backend is not implemented")

type Request struct {
	Project    string
	Repository string
	Key        string
	Paths      []string
	Workspace  string
	TTL        time.Duration
	MaxBytes   int64
}

type PurgeRequest struct {
	Project    string
	Repository string
}

type Store interface {
	Restore(ctx context.Context, request Request) (bool, error)
	Save(ctx context.Context, request Request) error
	Purge(ctx context.Context, request PurgeRequest) error
}

type LocalStore struct {
	Root string
	Now  func() time.Time
}

type ObjectStore struct{}

type manifest struct {
	SavedAt time.Time `json:"savedAt"`
}

func ScopeKey(project, repository, key string) (string, error) {
	if strings.TrimSpace(project) == "" || strings.TrimSpace(repository) == "" {
		return "", fmt.Errorf("project and repository are required")
	}
	if strings.TrimSpace(key) == "" {
		key = "default"
	}
	return filepath.Join(hash(project), hash(repository), hash(key)), nil
}

func RegistryRef(imageRepository, project, repository, key string) (string, error) {
	if strings.TrimSpace(imageRepository) == "" {
		return "", fmt.Errorf("image repository is required")
	}
	scope, err := ScopeKey(project, repository, key)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(scope))
	return fmt.Sprintf("%s:cache-%s", strings.TrimSuffix(imageRepository, ":"), hex.EncodeToString(digest[:8])), nil
}

func (s LocalStore) Restore(ctx context.Context, request Request) (bool, error) {
	entry, err := s.entry(request.Project, request.Repository, request.Key)
	if err != nil {
		return false, err
	}
	metadata, err := os.ReadFile(filepath.Join(entry, "manifest.json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read cache manifest: %w", err)
	}
	var stored manifest
	if err := json.Unmarshal(metadata, &stored); err != nil {
		return false, fmt.Errorf("decode cache manifest: %w", err)
	}
	if request.TTL > 0 && s.now().After(stored.SavedAt.Add(request.TTL)) {
		if err := os.RemoveAll(entry); err != nil {
			return false, fmt.Errorf("remove expired cache: %w", err)
		}
		return false, nil
	}
	for _, path := range request.Paths {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		clean, err := safeRelative(path)
		if err != nil {
			return false, err
		}
		source := filepath.Join(entry, "data", clean)
		if _, err := os.Lstat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return false, fmt.Errorf("inspect cached path %q: %w", path, err)
		}
		if err := copyTree(source, filepath.Join(request.Workspace, clean), nil); err != nil {
			return false, fmt.Errorf("restore cached path %q: %w", path, err)
		}
	}
	return true, nil
}

func (s LocalStore) Save(ctx context.Context, request Request) error {
	entry, err := s.entry(request.Project, request.Repository, request.Key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(entry), 0o750); err != nil {
		return fmt.Errorf("create cache scope: %w", err)
	}
	temporary, err := os.MkdirTemp(filepath.Dir(entry), ".cache-write-")
	if err != nil {
		return fmt.Errorf("create cache staging directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	var copied int64
	for _, path := range request.Paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		clean, err := safeRelative(path)
		if err != nil {
			return err
		}
		source := filepath.Join(request.Workspace, clean)
		if _, err := os.Lstat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return fmt.Errorf("inspect cache path %q: %w", path, err)
		}
		if err := copyTree(source, filepath.Join(temporary, "data", clean), func(size int64) error {
			copied += size
			if request.MaxBytes > 0 && copied > request.MaxBytes {
				return fmt.Errorf("cache exceeds configured maximum of %d bytes", request.MaxBytes)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("save cache path %q: %w", path, err)
		}
	}
	metadata, _ := json.Marshal(manifest{SavedAt: s.now().UTC()})
	if err := os.WriteFile(filepath.Join(temporary, "manifest.json"), metadata, 0o600); err != nil {
		return fmt.Errorf("write cache manifest: %w", err)
	}
	if err := os.RemoveAll(entry); err != nil {
		return fmt.Errorf("replace cache entry: %w", err)
	}
	if err := os.Rename(temporary, entry); err != nil {
		return fmt.Errorf("publish cache entry: %w", err)
	}
	return nil
}

func (s LocalStore) Purge(_ context.Context, request PurgeRequest) error {
	if strings.TrimSpace(request.Project) == "" {
		return fmt.Errorf("project is required")
	}
	target := filepath.Join(s.Root, hash(request.Project))
	if request.Repository != "" {
		target = filepath.Join(target, hash(request.Repository))
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("purge dependency cache: %w", err)
	}
	return nil
}

func (s LocalStore) entry(project, repository, key string) (string, error) {
	if strings.TrimSpace(s.Root) == "" {
		return "", fmt.Errorf("cache root is required")
	}
	scope, err := ScopeKey(project, repository, key)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.Root, scope), nil
}

func (s LocalStore) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (ObjectStore) Restore(context.Context, Request) (bool, error) {
	return false, ErrUnsupported
}
func (ObjectStore) Save(context.Context, Request) error       { return ErrUnsupported }
func (ObjectStore) Purge(context.Context, PurgeRequest) error { return ErrUnsupported }

func safeRelative(path string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("cache path %q must be a non-empty relative workspace path", path)
	}
	return clean, nil
}

func copyTree(source, destination string, observe func(int64) error) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic links are not cached")
	}
	if !info.IsDir() {
		return copyFile(source, destination, info.Mode(), observe)
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic links are not cached: %s", path)
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, fileInfo.Mode(), observe)
	})
}

func copyFile(source, destination string, mode fs.FileMode, observe func(int64) error) error {
	if observe != nil {
		info, err := os.Stat(source)
		if err != nil {
			return err
		}
		if err := observe(info.Size()); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm()&0o700)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:16])
}

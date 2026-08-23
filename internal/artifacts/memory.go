package artifacts

import (
	"context"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]Artifact
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{items: map[string]Artifact{}} }

func (s *MemoryStore) Put(_ context.Context, req PutArtifactRequest) (*ArtifactRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := ArtifactRef{Namespace: req.Namespace, BuildRun: req.BuildRun, Name: req.Name, Path: req.Path, Type: req.Type, Size: int64(len(req.Data)), Digest: req.Digest, Ref: "memory://" + req.Namespace + "/" + req.BuildRun + "/" + req.Name, CreatedAt: time.Now().UTC()}
	s.items[ref.Ref] = Artifact{Ref: ref, Data: append([]byte(nil), req.Data...)}
	return &ref, nil
}

func (s *MemoryStore) Get(_ context.Context, ref ArtifactRef) (*Artifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[ref.Ref]
	if !ok {
		return nil, ErrNotFound
	}
	item.Data = append([]byte(nil), item.Data...)
	return &item, nil
}

func (s *MemoryStore) List(_ context.Context, req ListArtifactsRequest) ([]ArtifactRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	refs := []ArtifactRef{}
	for _, item := range s.items {
		if item.Ref.Namespace == req.Namespace && item.Ref.BuildRun == req.BuildRun {
			refs = append(refs, item.Ref)
		}
	}
	return refs, nil
}

func (s *MemoryStore) Delete(_ context.Context, ref ArtifactRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, ref.Ref)
	return nil
}

package logstore

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu    sync.RWMutex
	lines map[string][]LogLine
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{lines: map[string][]LogLine{}} }

func (s *MemoryStore) Append(_ context.Context, req AppendLogRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := req.Namespace + "/" + req.BuildRun
	s.lines[key] = append(s.lines[key], req.Lines...)
	return nil
}

func (s *MemoryStore) Read(_ context.Context, req ReadLogRequest) (*ReadLogResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := req.Namespace + "/" + req.BuildRun
	lines, ok := s.lines[key]
	if !ok {
		return nil, ErrNotFound
	}
	return &ReadLogResult{Backend: "memory", Ref: key, Lines: FilterAndTail(append([]LogLine(nil), lines...), req.Step, req.TailLines)}, nil
}

func (s *MemoryStore) Stream(ctx context.Context, req StreamLogRequest) (<-chan LogLine, error) {
	result, err := s.Read(ctx, ReadLogRequest{Namespace: req.Namespace, BuildRun: req.BuildRun, TailLines: req.TailLines, Step: req.Step})
	if err != nil {
		return nil, err
	}
	return StreamSnapshot(ctx, result), nil
}

func (s *MemoryStore) Delete(_ context.Context, req DeleteLogRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.lines, req.Namespace+"/"+req.BuildRun)
	return nil
}

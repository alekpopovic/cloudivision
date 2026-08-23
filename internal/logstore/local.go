package logstore

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var safeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

type LocalStore struct{ Root string }

func (s LocalStore) path(namespace, buildRun string) (string, error) {
	if !safeName.MatchString(namespace) || !safeName.MatchString(buildRun) {
		return "", errors.New("namespace and BuildRun must be safe path components")
	}
	if s.Root == "" {
		return "", errors.New("local log root is required")
	}
	return filepath.Join(s.Root, namespace, buildRun+".jsonl"), nil
}

func (s LocalStore) Append(_ context.Context, req AppendLogRequest) error {
	path, err := s.path(req.Namespace, req.BuildRun)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, line := range req.Lines {
		if err := encoder.Encode(line); err != nil {
			return fmt.Errorf("append log line: %w", err)
		}
	}
	return nil
}

func (s LocalStore) Read(_ context.Context, req ReadLogRequest) (*ReadLogResult, error) {
	path, err := s.path(req.Namespace, req.BuildRun)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer file.Close()
	lines := []LogLine{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var line LogLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			return nil, fmt.Errorf("decode stored log line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read log file: %w", err)
	}
	return &ReadLogResult{Backend: "local", Ref: path, Lines: FilterAndTail(lines, req.Step, req.TailLines)}, nil
}

func (s LocalStore) Stream(ctx context.Context, req StreamLogRequest) (<-chan LogLine, error) {
	result, err := s.Read(ctx, ReadLogRequest{Namespace: req.Namespace, BuildRun: req.BuildRun, TailLines: req.TailLines, Step: req.Step})
	if err != nil {
		return nil, err
	}
	return StreamSnapshot(ctx, result), nil
}

func (s LocalStore) Delete(_ context.Context, req DeleteLogRequest) error {
	path, err := s.path(req.Namespace, req.BuildRun)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete log file: %w", err)
	}
	return nil
}

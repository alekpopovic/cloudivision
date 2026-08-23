package logstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound       = errors.New("logs not found")
	ErrNotImplemented = errors.New("log backend is not implemented")
)

type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Step      string    `json:"step,omitempty"`
	Message   string    `json:"message"`
}

type AppendLogRequest struct {
	Namespace string
	BuildRun  string
	Lines     []LogLine
}

type ReadLogRequest struct {
	Namespace string
	BuildRun  string
	TailLines int
	Step      string
}

type ReadLogResult struct {
	Backend string
	Ref     string
	Lines   []LogLine
}

type StreamLogRequest struct {
	Namespace string
	BuildRun  string
	TailLines int
	Step      string
	Follow    bool
}

type DeleteLogRequest struct {
	Namespace string
	BuildRun  string
}

type LogStore interface {
	Append(ctx context.Context, req AppendLogRequest) error
	Read(ctx context.Context, req ReadLogRequest) (*ReadLogResult, error)
	Stream(ctx context.Context, req StreamLogRequest) (<-chan LogLine, error)
	Delete(ctx context.Context, req DeleteLogRequest) error
}

type RedactingStore struct {
	Store LogStore
	Mask  func(string) string
}

func (s RedactingStore) Append(ctx context.Context, req AppendLogRequest) error {
	if s.Store == nil {
		return errors.New("log store is required")
	}
	if s.Mask != nil {
		for i := range req.Lines {
			req.Lines[i].Message = s.Mask(req.Lines[i].Message)
		}
	}
	return s.Store.Append(ctx, req)
}

func (s RedactingStore) Read(ctx context.Context, req ReadLogRequest) (*ReadLogResult, error) {
	return s.Store.Read(ctx, req)
}
func (s RedactingStore) Stream(ctx context.Context, req StreamLogRequest) (<-chan LogLine, error) {
	return s.Store.Stream(ctx, req)
}
func (s RedactingStore) Delete(ctx context.Context, req DeleteLogRequest) error {
	return s.Store.Delete(ctx, req)
}

type Writer struct {
	Context context.Context
	Store   LogStore
	Request AppendLogRequest
	Step    string
}

func (w Writer) Write(data []byte) (int, error) {
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	entries := make([]LogLine, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			entries = append(entries, LogLine{Timestamp: time.Now().UTC(), Step: w.Step, Message: line})
		}
	}
	if len(entries) == 0 {
		return len(data), nil
	}
	req := w.Request
	req.Lines = entries
	if err := w.Store.Append(w.Context, req); err != nil {
		return 0, fmt.Errorf("append stored logs: %w", err)
	}
	return len(data), nil
}

func FilterAndTail(lines []LogLine, step string, tail int) []LogLine {
	filtered := make([]LogLine, 0, len(lines))
	for _, line := range lines {
		if step == "" || line.Step == step {
			filtered = append(filtered, line)
		}
	}
	if tail > 0 && len(filtered) > tail {
		filtered = filtered[len(filtered)-tail:]
	}
	return filtered
}

func StreamSnapshot(ctx context.Context, result *ReadLogResult) <-chan LogLine {
	stream := make(chan LogLine)
	go func() {
		defer close(stream)
		for _, line := range result.Lines {
			select {
			case stream <- line:
			case <-ctx.Done():
				return
			}
		}
	}()
	return stream
}

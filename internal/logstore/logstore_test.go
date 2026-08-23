package logstore

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMemoryStoreTailAndStepFilter(t *testing.T) {
	store := NewMemoryStore()
	lines := []LogLine{
		{Timestamp: time.Unix(1, 0), Step: "test", Message: "one"},
		{Timestamp: time.Unix(2, 0), Step: "build", Message: "two"},
		{Timestamp: time.Unix(3, 0), Step: "test", Message: "three"},
	}
	if err := store.Append(context.Background(), AppendLogRequest{Namespace: "ci", BuildRun: "build-1", Lines: lines}); err != nil {
		t.Fatal(err)
	}
	result, err := store.Read(context.Background(), ReadLogRequest{Namespace: "ci", BuildRun: "build-1", Step: "test", TailLines: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Lines) != 1 || result.Lines[0].Message != "three" {
		t.Fatalf("lines = %#v", result.Lines)
	}
}

func TestLocalStorePersistsLogs(t *testing.T) {
	store := LocalStore{Root: t.TempDir()}
	req := AppendLogRequest{Namespace: "ci", BuildRun: "build-1", Lines: []LogLine{{Timestamp: time.Now(), Message: "persisted"}}}
	if err := store.Append(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	result, err := store.Read(context.Background(), ReadLogRequest{Namespace: "ci", BuildRun: "build-1"})
	if err != nil || len(result.Lines) != 1 || result.Lines[0].Message != "persisted" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestRedactingStoreMasksBeforeAppend(t *testing.T) {
	memory := NewMemoryStore()
	store := RedactingStore{Store: memory, Mask: func(value string) string { return strings.ReplaceAll(value, "secret-value", "[REDACTED]") }}
	if err := store.Append(context.Background(), AppendLogRequest{Namespace: "ci", BuildRun: "build-1", Lines: []LogLine{{Message: "token=secret-value"}}}); err != nil {
		t.Fatal(err)
	}
	result, err := memory.Read(context.Background(), ReadLogRequest{Namespace: "ci", BuildRun: "build-1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Lines[0].Message, "secret-value") || result.Lines[0].Message != "token=[REDACTED]" {
		t.Fatalf("message = %q", result.Lines[0].Message)
	}
}

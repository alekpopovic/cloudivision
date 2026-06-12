package audit

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestNoopRecorder(t *testing.T) {
	if err := (NoopRecorder{}).Record(context.Background(), Event{Type: "test"}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
}

func TestLogRecorderRedactsSecrets(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	recorder := LogRecorder{Logger: logger}

	err := recorder.Record(context.Background(), Event{
		Type:    "Webhook",
		Actor:   "token=secret-token",
		Message: "password=hunter2 normal=visible",
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	got := output.String()
	if strings.Contains(got, "secret-token") || strings.Contains(got, "hunter2") {
		t.Fatalf("log output leaked secret: %s", got)
	}
	if !strings.Contains(got, "visible") {
		t.Fatalf("log output = %s, want visible non-secret value", got)
	}
}

func TestPostgresRecorderRecordsAndListsEvents(t *testing.T) {
	databaseURL := os.Getenv("CLOU_DIVISION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("CLOU_DIVISION_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	applyTestSchema(t, ctx, db)
	recorder := NewPostgresRecorder(db)
	eventID := "test-" + time.Now().UTC().Format("20060102150405.000000000")

	err = recorder.Record(ctx, Event{
		ID:         eventID,
		Type:       "BuildRunCreated",
		Actor:      "tester",
		Project:    "project",
		Repository: "repo",
		BuildRun:   "build",
		Message:    "created",
		Metadata:   json.RawMessage(`{"source":"test"}`),
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	events, err := recorder.ListEvents(ctx, EventFilter{Project: "project", BuildRun: "build", Type: "BuildRunCreated"})
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("ListEvents() returned no events")
	}
}

func TestPostgresWebhookIndex(t *testing.T) {
	databaseURL := os.Getenv("CLOU_DIVISION_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("CLOU_DIVISION_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	applyTestSchema(t, ctx, db)
	recorder := NewPostgresRecorder(db)
	eventID := "event-" + time.Now().UTC().Format("20060102150405.000000000")

	if err := recorder.RecordWebhookEvent(ctx, WebhookEvent{
		Provider:   "github",
		Repository: "repo",
		EventID:    eventID,
		Project:    "project",
		BuildRun:   "build",
	}); err != nil {
		t.Fatalf("RecordWebhookEvent() error = %v", err)
	}
	found, err := recorder.FindWebhookEvent(ctx, "github", "repo", eventID)
	if err != nil {
		t.Fatalf("FindWebhookEvent() error = %v", err)
	}
	if found == nil || found.BuildRun != "build" {
		t.Fatalf("found = %#v, want build", found)
	}
}

func applyTestSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, path := range []string{
		"migrations/0001_audit_events.sql",
		"migrations/0002_webhook_events.sql",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := db.ExecContext(ctx, string(data)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
}

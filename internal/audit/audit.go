package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudivision/cloudivision/internal/redact"
)

type Event struct {
	ID         string          `json:"id,omitempty"`
	Type       string          `json:"type"`
	Actor      string          `json:"actor,omitempty"`
	Project    string          `json:"project,omitempty"`
	Repository string          `json:"repository,omitempty"`
	BuildRun   string          `json:"buildRun,omitempty"`
	Release    string          `json:"release,omitempty"`
	Message    string          `json:"message,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"createdAt,omitempty"`
	EventID    string          `json:"eventID,omitempty"`
}

type EventFilter struct {
	Project  string
	BuildRun string
	Release  string
	Type     string
}

type Recorder interface {
	Record(ctx context.Context, event Event) error
}

type EventLister interface {
	ListEvents(ctx context.Context, filter EventFilter) ([]Event, error)
}

type WebhookEvent struct {
	Provider   string
	Repository string
	EventID    string
	Project    string
	BuildRun   string
	CreatedAt  time.Time
}

type WebhookIndexer interface {
	FindWebhookEvent(ctx context.Context, provider, repository, eventID string) (*WebhookEvent, error)
	RecordWebhookEvent(ctx context.Context, event WebhookEvent) error
}

type NoopRecorder struct{}

func (NoopRecorder) Record(context.Context, Event) error {
	return nil
}

type LogRecorder struct {
	Logger *slog.Logger
}

type LoggerRecorder = LogRecorder

func (r LogRecorder) Record(_ context.Context, event Event) error {
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("audit event",
		"id", redact.MaskString(event.ID),
		"type", redact.MaskString(event.Type),
		"actor", redact.MaskString(event.Actor),
		"project", redact.MaskString(event.Project),
		"repository", redact.MaskString(event.Repository),
		"buildRun", redact.MaskString(event.BuildRun),
		"release", redact.MaskString(event.Release),
		"eventID", redact.MaskString(event.EventID),
		"message", redact.MaskString(event.Message),
	)
	return nil
}

type PostgresRecorder struct {
	DB *sql.DB
}

func NewPostgresRecorder(db *sql.DB) PostgresRecorder {
	return PostgresRecorder{DB: db}
}

func (r PostgresRecorder) Record(ctx context.Context, event Event) error {
	if r.DB == nil {
		return errors.New("postgres audit recorder requires a database")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = newID()
	}
	metadata := []byte(event.Metadata)
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	_, err := r.DB.ExecContext(ctx, `
insert into audit_events (
  id, type, actor, project, repository, build_run, release, message, metadata, created_at
) values (
  $1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10
)`,
		event.ID,
		event.Type,
		nullString(event.Actor),
		nullString(event.Project),
		nullString(event.Repository),
		nullString(event.BuildRun),
		nullString(event.Release),
		nullString(event.Message),
		string(metadata),
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func (r PostgresRecorder) ListEvents(ctx context.Context, filter EventFilter) ([]Event, error) {
	if r.DB == nil {
		return nil, errors.New("postgres audit lister requires a database")
	}
	rows, err := r.DB.QueryContext(ctx, `
select id, type, coalesce(actor, ''), coalesce(project, ''), coalesce(repository, ''),
       coalesce(build_run, ''), coalesce(release, ''), coalesce(message, ''),
       metadata, created_at
from audit_events
where ($1 = '' or project = $1)
  and ($2 = '' or build_run = $2)
  and ($3 = '' or release = $3)
  and ($4 = '' or type = $4)
order by created_at desc
limit 200`,
		filter.Project,
		filter.BuildRun,
		filter.Release,
		filter.Type,
	)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var event Event
		var metadata []byte
		if err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Actor,
			&event.Project,
			&event.Repository,
			&event.BuildRun,
			&event.Release,
			&event.Message,
			&metadata,
			&event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		event.Metadata = json.RawMessage(metadata)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return events, nil
}

func (r PostgresRecorder) FindWebhookEvent(ctx context.Context, provider, repository, eventID string) (*WebhookEvent, error) {
	if r.DB == nil {
		return nil, errors.New("postgres webhook index requires a database")
	}
	row := r.DB.QueryRowContext(ctx, `
select provider, repository, event_id, coalesce(project, ''), build_run, created_at
from webhook_events
where provider = $1 and repository = $2 and event_id = $3`,
		provider,
		repository,
		eventID,
	)
	var event WebhookEvent
	if err := row.Scan(&event.Provider, &event.Repository, &event.EventID, &event.Project, &event.BuildRun, &event.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find webhook event: %w", err)
	}
	return &event, nil
}

func (r PostgresRecorder) RecordWebhookEvent(ctx context.Context, event WebhookEvent) error {
	if r.DB == nil {
		return errors.New("postgres webhook index requires a database")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	_, err := r.DB.ExecContext(ctx, `
insert into webhook_events (provider, repository, event_id, project, build_run, created_at)
values ($1, $2, $3, $4, $5, $6)
on conflict (provider, repository, event_id) do nothing`,
		event.Provider,
		event.Repository,
		event.EventID,
		nullString(event.Project),
		event.BuildRun,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("record webhook event: %w", err)
	}
	return nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func newID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("audit-%d", time.Now().UTC().UnixNano())
	}
	return hex.EncodeToString(bytes[:])
}

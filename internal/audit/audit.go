package audit

import (
	"context"
	"log/slog"
)

type Event struct {
	Type    string
	Actor   string
	Subject string
	Message string
	EventID string
}

type Recorder interface {
	Record(ctx context.Context, event Event) error
}

type LoggerRecorder struct {
	Logger *slog.Logger
}

func (r LoggerRecorder) Record(_ context.Context, event Event) error {
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("audit event",
		"type", event.Type,
		"actor", event.Actor,
		"subject", event.Subject,
		"eventID", event.EventID,
		"message", event.Message,
	)
	return nil
}

package audit

import (
	"context"
	"log/slog"

	"github.com/cloudivision/cloudivision/internal/redact"
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
		"type", redact.MaskString(event.Type),
		"actor", redact.MaskString(event.Actor),
		"subject", redact.MaskString(event.Subject),
		"eventID", redact.MaskString(event.EventID),
		"message", redact.MaskString(event.Message),
	)
	return nil
}

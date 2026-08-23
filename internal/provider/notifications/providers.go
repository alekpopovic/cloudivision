package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudivision/cloudivision/internal/provider"
)

var ErrUnsupported = errors.New("notification provider is not configured")

type Event string

const (
	BuildRunStarted         Event = "BuildRunStarted"
	BuildRunSucceeded       Event = "BuildRunSucceeded"
	BuildRunFailed          Event = "BuildRunFailed"
	ReleaseAwaitingApproval Event = "ReleaseAwaitingApproval"
	ReleaseDeployed         Event = "ReleaseDeployed"
	ReleaseFailed           Event = "ReleaseFailed"
	PolicyDenied            Event = "PolicyDenied"
	WebhookRejected         Event = "WebhookRejected"
)

type NotificationRequest struct {
	Event        Event     `json:"event"`
	Project      string    `json:"project,omitempty"`
	Repository   string    `json:"repository,omitempty"`
	Environment  string    `json:"environment,omitempty"`
	Phase        string    `json:"phase,omitempty"`
	Namespace    string    `json:"namespace,omitempty"`
	ResourceName string    `json:"resourceName,omitempty"`
	Message      string    `json:"message,omitempty"`
	OccurredAt   time.Time `json:"occurredAt"`
}

type NotificationProvider interface {
	Send(ctx context.Context, req NotificationRequest) error
	HealthCheck(ctx context.Context) provider.ProviderHealth
}

type Dispatcher interface {
	Notify(ctx context.Context, req NotificationRequest) error
}

type GenericWebhook struct {
	Endpoint    string
	Client      *http.Client
	MaxAttempts int
	Backoff     time.Duration
}

func (p GenericWebhook) Name() string { return "webhook" }
func (p GenericWebhook) Type() string { return "notifications" }
func (p GenericWebhook) Capabilities() []provider.Capability {
	return []provider.Capability{{Name: "send", Description: "Send redacted JSON event payloads to a generic webhook"}}
}
func (p GenericWebhook) HealthCheck(context.Context) provider.ProviderHealth {
	return provider.ProviderHealth{Healthy: true, Message: "webhook endpoints are supplied by Project secrets", CheckedAt: time.Now().UTC()}
}
func (p GenericWebhook) Send(ctx context.Context, request NotificationRequest) error {
	if p.Endpoint == "" {
		return fmt.Errorf("send notification: %w", ErrUnsupported)
	}
	if request.OccurredAt.IsZero() {
		request.OccurredAt = time.Now().UTC()
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encode notification payload: %w", err)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	attempts := p.MaxAttempts
	if attempts < 1 {
		attempts = 3
	}
	backoff := p.Backoff
	if backoff <= 0 {
		backoff = 100 * time.Millisecond
	}
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(payload))
		if err != nil {
			return errors.New("create notification request")
		}
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return nil
			}
			if response.StatusCode < 500 {
				return fmt.Errorf("notification webhook returned HTTP %d", response.StatusCode)
			}
		}
		if attempt < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff * time.Duration(attempt)):
			}
		}
	}
	return errors.New("notification webhook delivery failed after retries")
}

type Skeleton struct{ ProviderName string }

func (p Skeleton) Name() string { return p.ProviderName }
func (p Skeleton) Type() string { return "notifications" }
func (p Skeleton) Capabilities() []provider.Capability {
	return []provider.Capability{{Name: "skeleton", Description: "Configuration and health skeleton; delivery is not implemented"}}
}
func (p Skeleton) HealthCheck(context.Context) provider.ProviderHealth {
	return provider.ProviderHealth{Healthy: false, Message: p.ProviderName + " notification delivery is a skeleton", CheckedAt: time.Now().UTC()}
}
func (p Skeleton) Send(context.Context, NotificationRequest) error {
	return fmt.Errorf("%s: %w", p.ProviderName, ErrUnsupported)
}

func Noop() provider.Provider {
	return provider.Static{ProviderName: "noop", ProviderType: "notifications", Healthy: true, Message: "notifications are disabled", Features: []provider.Capability{{Name: "discard", Description: "Accept notification events without external delivery"}}}
}
func Webhook() provider.Provider { return GenericWebhook{} }
func Slack() provider.Provider   { return Skeleton{ProviderName: "slack"} }
func Teams() provider.Provider   { return Skeleton{ProviderName: "teams"} }
func Email() provider.Provider   { return Skeleton{ProviderName: "email"} }

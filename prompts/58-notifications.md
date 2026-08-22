# 58. Notifications

```text
Continue working on the cloudivision project.

Task:
Add notification provider support.

Goal:
Notify teams about important build and release events.

Providers:
- webhook generic
- Slack skeleton
- Microsoft Teams skeleton
- email skeleton

Events:
- BuildRunStarted
- BuildRunSucceeded
- BuildRunFailed
- ReleaseAwaitingApproval
- ReleaseDeployed
- ReleaseFailed
- PolicyDenied
- WebhookRejected

CRD/API:
Add NotificationConfig resource or Project.spec.notifications:
- enabled bool
- provider
- secretRef
- events []
- filters:
  - project
  - repository
  - environment
  - phase

Provider interface:
type NotificationProvider interface {
    Send(ctx context.Context, req NotificationRequest) error
    HealthCheck(ctx context.Context) ProviderHealth
}

Security:
- Do not log webhook URLs or tokens.
- Redact notification secrets.
- Do not send secrets in notification payloads.

Angular UI:
- Project notifications settings page.
- Provider health display.

Tests:
- generic webhook payload
- event filtering
- secret redaction
- provider failure does not break BuildRun controller
- retry/backoff skeleton if needed

Docs:
- docs/notifications/index.md
- docs/notifications/webhook.md
- docs/notifications/slack.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Notification failures are visible but do not crash controllers.
- Generic webhook provider works or has clear skeleton behavior.
```

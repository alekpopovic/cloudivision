# Notifications

Projects can route important BuildRun, Release, policy and webhook events to a notification provider. Configure `spec.notifications.enabled`, `provider`, a Secret key reference, optional `events`, and project/repository/environment/phase filters.

Supported events are `BuildRunStarted`, `BuildRunSucceeded`, `BuildRunFailed`, `ReleaseAwaitingApproval`, `ReleaseDeployed`, `ReleaseFailed`, `PolicyDenied`, and `WebhookRejected`. Delivery failures add a `NotificationDelivered=False` condition but never fail the build or release controller.

Only the generic webhook provider sends messages today. Slack, Microsoft Teams and email expose explicit health/capability skeletons. Provider health is visible through `/api/v1/providers/health` and on the Project detail settings section.

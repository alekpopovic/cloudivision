# Webhook security

cloudivision reads a Repository Secret, verifies provider authentication before parsing, requires an event ID and commit SHA, enforces the configured default branch, and suppresses duplicate events through PostgreSQL indexing or existing BuildRun lookup.

```yaml
spec:
  provider: github
  webhook:
    enabled: true
    secretRef: {name: app-webhook, key: secret}
    events: [push]
```

Use TLS ingress, request-size/rate limits, provider IP controls where operationally reliable, random per-repository secrets, coordinated rotation, and bounded delivery retention. Monitor `cloudivision_webhook_events_total{outcome="invalid_signature"}`. Never log payloads/signatures/tokens.

Current limitations: branch filtering is the single default branch rather than configurable patterns; event-time replay windows are not enforced independently of event ID; GitHub/GitLab/Gitea support depth differs; pull-request events do not trigger distinct pipeline policy yet; in-memory/Kubernetes fallback idempotency is less durable than PostgreSQL across pruning.

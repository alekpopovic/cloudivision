# Webhook security

The GitHub webhook endpoint is:

```text
POST /api/v1/webhooks/github/{repositoryName}?namespace={namespace}
```

It is intentionally outside user authentication. The shared webhook secret is
the authentication boundary, so expose the endpoint only over HTTPS and use a
unique, randomly generated secret for each Repository.

## Request validation

For an enabled GitHub webhook, cloudivision performs these checks before creating
a BuildRun:

1. Reads at most 1 MiB. Larger requests return HTTP 413 with
   `payload_too_large`.
2. Loads the exact Secret and key from `Repository.spec.webhook.secretRef`.
3. verifies `X-Hub-Signature-256` as HMAC-SHA256 over the unmodified request
   bytes using constant-time comparison. Missing or invalid signatures return
   HTTP 401.
4. Parses the GitHub event and requires `X-GitHub-Delivery` (maximum 255
   characters).
5. When GitHub supplies `repository.pushed_at` for a push or `updated_at` for a
   pull request, rejects events older than 24 hours or more than five minutes in
   the future. Commit author timestamps are not used as delivery timestamps.
6. Checks the delivery ID in the webhook idempotency backend, with a Kubernetes
   BuildRun lookup as a fallback for build-producing events.

The endpoint handles `push`, `pull_request`, and `ping`. Pull requests create a
BuildRun for `opened`, `reopened`, and `synchronize` actions when their base branch
matches the Repository `defaultBranch`. A ping succeeds without a BuildRun.

## Idempotency and audit

Use `CLOU_DIVISION_AUDIT_BACKEND=postgres` in a multi-replica production API.
Apply every SQL migration in `internal/audit/migrations` before enabling it. The
PostgreSQL primary key on provider, Repository, and delivery ID stores processed
deliveries and makes sequential redelivery deterministic. BuildRun names are also
derived from delivery IDs, so a concurrent create cannot produce two BuildRuns.

Bounded audit records use these event types:

- `WebhookAccepted`
- `WebhookRejected`
- `WebhookDuplicate`
- `BuildRunCreatedFromWebhook`

Audit metadata contains only provider, event type, delivery ID, and a stable
reason. cloudivision does not log or persist the full webhook payload or shared
secret. Retain audit records according to your incident-response policy and alert
on repeated `WebhookRejected` events.

## Secret rotation

GitHub supports one webhook secret at a time. To rotate it:

1. Generate a new random secret.
2. Update the GitHub webhook secret and the referenced Kubernetes Secret in a
   short maintenance window.
3. Trigger GitHub's **Redeliver** action and confirm a signed ping or push is
   accepted.
4. Remove all local copies of both values.

During the brief interval between updates, deliveries signed with the mismatched
value are safely rejected and can be redelivered from GitHub afterward.

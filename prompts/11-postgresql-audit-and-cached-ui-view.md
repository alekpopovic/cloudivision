# Prompt 11 — PostgreSQL Audit and Cached UI View

Phase: Audit

```text
Continue working on the cloudivision project.

Task:
Add PostgreSQL storage for audit events and optional cached UI views, but do not move Kubernetes runtime state out of CRD status.

Goal:
The database is used for audit events, webhook event idempotency index, cached list views for UI if enabled and future users/teams/auth model.

Steps:
1. Add /internal/audit package with Event struct fields: ID, Type, Actor, Project, Repository, BuildRun, Release, Message, Metadata JSON, CreatedAt.
2. Add Recorder interface: Record(ctx, Event) error.
3. Implement NoopRecorder, LogRecorder and PostgresRecorder.
4. Add migrations for audit_events and webhook_events.
5. Add config: CLOU_DIVISION_DATABASE_URL and CLOU_DIVISION_AUDIT_BACKEND=noop|log|postgres.
6. Webhook idempotency uses webhook_events table if postgres is configured; fallback to Kubernetes label/annotation lookup if postgres is not enabled.
7. Add GET /api/v1/audit/events with filters project, buildRun, release and type.
8. Add tests for recorders and idempotency fallback. Postgres tests should cleanly skip if test DB is unavailable.

Acceptance criteria:
- go test ./... passes.
- Application works without PostgreSQL.
- If PostgreSQL is not configured, no fatal error unless config explicitly requires postgres.
- Runtime status for BuildRun and Release still comes from Kubernetes CRD status.
- Migrations are versioned.
```

# Database storage

PostgreSQL is optional in the MVP. Kubernetes CRD status remains the source of truth for BuildRun and
Release runtime state.

The database is used for audit events, webhook event idempotency, optional cached UI views, and future
users/teams/auth data.

Configuration:

- `CLOU_DIVISION_AUDIT_BACKEND=noop|log|postgres`
- `CLOU_DIVISION_DATABASE_URL=postgres://...`

`CLOU_DIVISION_AUDIT_BACKEND` defaults to `log`, so the API starts without PostgreSQL. If the backend
is set to `postgres`, `CLOU_DIVISION_DATABASE_URL` is required and the API verifies the connection on
startup.

Versioned migrations live in `/internal/audit/migrations`.

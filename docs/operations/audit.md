# Audit operations and export

Audit events are emitted for build/release actions, approvals, webhooks and
policy decisions. Kubernetes resource status remains authoritative for runtime
state; the optional PostgreSQL backend provides durable query/export history.
Apply all files under `internal/audit/migrations`, including the organization
column migration, before starting a PostgreSQL-backed API version that uses it.

`GET /api/v1/audit/events/export` returns JSON by default or CSV with
`format=csv`. Supported filters are `organization`, `project`, `repository`,
`buildRun`, `release`, `actor`, `eventType` (or `type`), and RFC3339 `from`/`to`.
Example:

```bash
curl -fsS -H "Authorization: Bearer $TOKEN" \
  'https://cloudivision.example/api/v1/audit/events/export?organization=acme&project=store&format=csv' \
  -o audit-events.csv
```

Exports require audit permission (`auditor`, `project-admin`, `org-admin`, or
global `admin`, subject to project grants). Metadata is recursively redacted
when a key resembles a token, password, secret, authorization value, private
key, or client secret. Do not treat export redaction as permission isolation:
scope every request to the tenant and retain files according to policy.

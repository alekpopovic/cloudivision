# Compliance reports

The API and Angular Reports page provide JSON summaries and CSV downloads:

- `GET /api/v1/reports/builds` — totals, success/failure rates, average completed
  duration and failures by reason;
- `GET /api/v1/reports/releases` — releases by environment, approval/rejection
  history, rollbacks and deployment failures;
- `GET /api/v1/reports/security` — policy denials, unsigned-release blocks,
  webhook rejections and critical-vulnerability blocks.

All endpoints accept `organization`, `project`, `repository`, resource name,
actor/event filters where applicable, plus RFC3339 `from` and `to`. Add
`format=csv` for a download. Reports combine current Kubernetes CR status with
durable audit history, so totals can differ during retention gaps or while the
audit backend is unavailable. Empty history returns zero counts, not fabricated
events.

Reports contain identifiers, actor identity and operational outcomes but never
Secret objects or raw credentials. Access uses the same audit-export permission
and organization/project grants as detailed audit export.

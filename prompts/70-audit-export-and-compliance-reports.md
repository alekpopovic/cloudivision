# 70. Audit Export and Compliance Reports

```text
Continue working on the cloudivision project.

Task:
Add audit export and basic compliance reporting.

Goal:
Users should be able to export build/release/security history.

API:
- GET /api/v1/audit/events/export
- GET /api/v1/reports/builds
- GET /api/v1/reports/releases
- GET /api/v1/reports/security

Formats:
- JSON
- CSV

Filters:
- organization
- project
- repository
- buildRun
- release
- actor
- event type
- from/to time

Reports:
1. Build report
- total builds
- success/failure rate
- average duration
- failed builds by reason

2. Release report
- releases by environment
- approval history
- rollback history
- deployment failures

3. Security report
- policy denials
- unsigned releases blocked
- webhook rejections
- critical vulnerability blocks if scanner exists

Angular UI:
- Audit export page
- Reports page

Tests:
- export JSON
- export CSV
- filters
- permission checks
- no secret leakage

Docs:
- docs/operations/audit.md
- docs/reports/index.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Audit export works.
- Reports do not expose secrets.
- Permission checks protect exports.
```

# 71. OpenAPI, SDK and API Client Generation

```text
Continue working on the cloudivision project.

Task:
Improve API documentation and generate clients.

Goal:
Make cloudivision API easier to consume by CLI, UI and external integrations.

OpenAPI:
- Ensure docs/openapi.yaml is complete.
- Include:
  - auth
  - error schema
  - pagination schema
  - BuildRun actions
  - Release actions
  - providers
  - audit
  - reports if implemented

Generation:
- Generate TypeScript client for Angular if practical.
- Generate Go client for CLI if practical.
- Keep generated code in a clear folder:
  - /sdk/typescript
  - /sdk/go
  - or /internal/generated if only internal

Rules:
- Do not manually duplicate API types unnecessarily.
- Keep generated clients updated through Makefile.
- Add CI check for OpenAPI generation.

Makefile:
- openapi-generate
- sdk-generate
- sdk-check

Tests:
- generated client compiles
- Angular can use generated types or migration plan exists
- CLI can use Go client or migration plan exists

Docs:
- docs/reference/api.md
- docs/development/api-client-generation.md

Acceptance criteria:
- OpenAPI spec is updated.
- SDK generation path exists.
- API error format is documented.
- go test ./... passes.
- npm run build passes in /web.
```

# Prompt 06 — API Server: REST API Over CRDs

Phase: API

```text
Continue working on the cloudivision project.

Task:
Implement the Go API server for cloudivision.

The API server should be a thin layer over the Kubernetes API and cloudivision CRDs.

MVP endpoints:
- GET /healthz
- GET /readyz
- GET /api/v1/projects
- POST /api/v1/projects
- GET /api/v1/projects/{name}
- GET /api/v1/repositories
- POST /api/v1/repositories
- GET /api/v1/pipeline-templates
- POST /api/v1/pipeline-templates
- GET /api/v1/build-runs
- POST /api/v1/build-runs
- GET /api/v1/build-runs/{namespace}/{name}
- GET /api/v1/build-runs/{namespace}/{name}/logs
- GET /api/v1/environments
- GET /api/v1/releases

Implementation:
1. Use Go net/http or a lightweight router already chosen in the repository.
2. Prefer minimal dependencies.
3. Use controller-runtime/client or client-go.
4. Do not duplicate runtime state in a database.
5. POST /build-runs creates a BuildRun CR.
6. GET /build-runs reads BuildRun CRDs and returns status.
7. Logs endpoint finds Pod by label cloudivision.io/buildrun and streams or returns tail lines; support query param tailLines.
8. Add request/response DTOs separate from CRD types.
9. Add basic input validation.
10. Add structured logging.
11. Add /docs/openapi.yaml or generated OpenAPI.
12. Add configurable CORS for local Angular development.

Auth:
- Placeholder middleware:
  - CLOU_DIVISION_AUTH_MODE=disabled allows all requests.
  - non-disabled returns 501 Not Implemented with a clear message.
- Do not implement fake production-looking security.

Acceptance criteria:
- go test ./... passes.
- API server can start locally.
- POST /api/v1/build-runs creates a valid BuildRun.
- GET /logs reads Kubernetes Pod logs when they exist.
- Error responses are JSON and contain code/message.
- API does not hardcode namespace.
```

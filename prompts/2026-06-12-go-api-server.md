# Go API server

User prompt:

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
1. Use Go standard net/http or a lightweight router if already chosen in the repository.
2. Prefer minimal dependencies.
3. API server uses controller-runtime/client or client-go.
4. API server must not duplicate runtime state in a database.
5. POST /build-runs creates a BuildRun CR.
6. GET /build-runs reads BuildRun CRDs and returns status.
7. Logs endpoint:
   - find Pod by label cloudivision.io/buildrun
   - stream logs or return last lines
   - support query param tailLines
8. Add request/response DTO models separate from CRD types.
9. Add basic input validation.
10. Add structured logging.
11. Add /docs/openapi.yaml or generate OpenAPI if tooling already exists.
12. Add CORS configuration suitable for local Angular development, but keep it configurable.

Auth:
- Implement placeholder middleware:
  - if CLOU_DIVISION_AUTH_MODE=disabled, allow all requests
  - if not disabled, return 501 Not Implemented with a clear message
- Do not implement fake security that looks production-ready but is not.

Acceptance criteria:
- go test ./... passes.
- API server can start locally.
- POST /api/v1/build-runs creates a valid BuildRun.
- GET /logs reads Kubernetes Pod logs when they exist.
- Error responses are JSON and contain code/message.
- API does not hardcode a namespace; use request input or configured default namespace.
```

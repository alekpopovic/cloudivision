# Prompt 15 — Observability: Metrics, Events, Tracing-Ready Logs

Phase: Observability

```text
Continue working on the cloudivision project.

Task:
Add observability for controller, API, runner and Angular UI.

Controller:
- Prometheus metrics: cloudivision_buildrun_total{phase}, cloudivision_buildrun_duration_seconds, cloudivision_reconcile_errors_total{controller}, cloudivision_reconcile_duration_seconds{controller}.
- Kubernetes Events for important transitions.
- Structured logs with project, buildRun, release, namespace and correlation ID when available.

API:
- HTTP request count, latency and error metrics.
- Request ID middleware.
- Access log without secret values.
- /metrics endpoint if enabled.
- Include request ID in JSON error responses.

Runner:
- Log every step started/completed/duration/exit code.
- Never log secret env values.
- Update BuildRun conditions.

Angular UI:
- Display backend request IDs in error details when provided.
- Add ErrorMessage component showing code, message and request ID.
- Do not log secrets to browser console.
- Avoid noisy console logs in production builds.

Tracing:
- Full OpenTelemetry implementation is not required now.
- Prepare middleware/interceptor extension points for future tracing.
- Add clear design TODOs only where there is a real planned integration.

Acceptance criteria:
- go test ./... passes.
- npm run build passes for /web.
- /metrics works for API and controller if enabled.
- Metrics do not use high-cardinality labels like commit SHA.
- Logs include request/buildrun correlation.
- Secret redaction is used in log paths.
- Angular UI surfaces useful API errors.
```

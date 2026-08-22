# Tracing preparation

cloudivision does not yet initialize an OpenTelemetry SDK or export spans. It already propagates a request ID through API logging and stores `cloudivision.io/correlation-id` on created resources where applicable. Keep this correlation path stable while tracing is introduced.

The future trace should connect these spans:

1. `api.request` or `webhook.receive` accepts W3C `traceparent`, verifies authentication/signature, and creates a BuildRun.
2. `buildrun.create` records the Kubernetes create call and writes trace context to safe annotations.
3. `controller.buildrun.reconcile` links asynchronous reconciliation to that context and creates `executor.ensure_run`.
4. `runner.execute` creates a child span per pipeline step, plus checkout, build, SBOM, scan, signing, and provenance spans.
5. `controller.release.reconcile` and `gitops.update` cover promotion, commit/push or PR creation, and provider health polling.

Because Kubernetes reconciliation is asynchronous, use span links when a strict parent span has ended. Propagate trace context through annotations and runner environment variables, with explicit size and character validation. Do not place trace context in labels.

Recommended low-cardinality attributes include component, controller, executor type, trigger type, policy result, provider type, operation, phase, and outcome. Resource namespace/name, request ID, BuildRun/Release UID, and commit SHA may be trace attributes because traces are sampled and queried differently from metrics, but define retention and access controls first. Never record secret values, tokens, webhook signatures, full authenticated repository URLs, arbitrary command output, actor email, or raw request bodies.

Adopt OpenTelemetry in stages: SDK/resource configuration, inbound HTTP instrumentation, Kubernetes client spans, controller span links, runner propagation, Git/GitOps spans, then tail-based sampling for failures and slow operations. Keep Prometheus metrics and structured logs as independent signals; tracing must not be required for reconciliation correctness.

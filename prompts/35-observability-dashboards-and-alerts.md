# Prompt 35 — Observability Dashboards and Alerts

Phase: Observability

```text
Continue working on the cloudivision project.

Task:
Add operational observability assets for cloudivision.

Goal:
Operators should be able to monitor cloudivision health, build throughput, failures and deployment problems.

Create:
- /deploy/observability/grafana-dashboard.json
- /deploy/observability/prometheus-rules.yaml
- /docs/operations/observability.md

Grafana dashboard panels:
1. BuildRuns by phase
2. BuildRun success/failure rate
3. Build duration p50/p95/p99
4. Build queue time if metric exists
5. Runner Job failures
6. Runner image pull failures if available
7. Controller reconcile error rate
8. Controller reconcile duration
9. API request rate
10. API latency p95/p99
11. API 5xx rate
12. Webhook accepted/invalid/replayed counts
13. Release phase distribution
14. GitOps push failures
15. Release deployment duration
16. PostgreSQL audit write failures if audit postgres enabled

Prometheus alert rules:
- controller reconcile errors high
- BuildRun stuck Running longer than configured timeout
- Release stuck Deploying longer than configured timeout
- API 5xx rate high
- webhook invalid signature spike
- runner Job failures high
- GitOps push failures
- audit database errors

Metrics review:
- Avoid high-cardinality labels such as commit SHA, full repository URL or actor email.
- Use project/repository labels only if cardinality is acceptable and documented.
- Add comments where metrics are intentionally limited.

OpenTelemetry preparation:
- Add docs/operations/tracing.md.
- Document future tracing path across API request, BuildRun create, controller reconcile, runner step and GitOps release spans.

Acceptance criteria:
- Grafana dashboard JSON exists.
- Prometheus rules exist.
- Observability docs exist.
- Existing metrics are documented.
- High-cardinality labels are avoided or justified.
- go test ./... passes.
```

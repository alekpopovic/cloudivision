# Observability

cloudivision exposes Prometheus metrics from the controller on port 8080 and from the API at `/metrics` when `api.metrics.enabled=true`. The Helm chart creates `cloudivision-controller-metrics` and `cloudivision-api` Services; configure your Prometheus installation with a ServiceMonitor or equivalent scrape configuration for both. The chart does not install Prometheus, Grafana, kube-state-metrics, or the Prometheus Operator.

Import `deploy/observability/grafana-dashboard.json` into Grafana and select the Prometheus datasource. Apply `deploy/observability/prometheus-rules.yaml` only when the `monitoring.coreos.com/v1` PrometheusRule CRD is installed:

```sh
kubectl apply -f deploy/observability/prometheus-rules.yaml
```

Runner image-pull and stuck-BuildRun panels/alerts use standard kube-state-metrics Job, Pod, and label metrics. Ensure kube-state-metrics exports `kube_job_status_start_time`, `kube_job_status_active`, `kube_job_labels`, `kube_pod_container_status_waiting_reason`, and `kube_pod_labels`.

## Metrics inventory

| Metric | Type | Purpose |
|---|---|---|
| `cloudivision_buildrun_total{phase}` | counter | BuildRun status updates by bounded phase; use rates, not as a current object count |
| `cloudivision_buildrun_duration_seconds` | histogram | completed build duration |
| `cloudivision_buildrun_queue_duration_seconds` | histogram | creation-to-Running queue time |
| `cloudivision_runner_job_failures_total` | counter | Job executor terminal failures |
| `cloudivision_release_status_updates_total{phase}` | counter | Release status updates by bounded phase |
| `cloudivision_release_deployment_duration_seconds` | histogram | GitOps preparation to terminal release duration |
| `cloudivision_release_in_progress_started_timestamp_seconds{namespace,name,phase}` | gauge | actionable, in-progress release start time for stuck alerts |
| `cloudivision_gitops_failures_total{operation}` | counter | clone, commit, push, pull-request, or provider-status failures |
| `cloudivision_reconcile_errors_total{controller}` | counter | reconcile errors |
| `cloudivision_reconcile_duration_seconds{controller}` | histogram | reconcile latency |
| `cloudivision_http_requests_total{method,route,status}` | counter | API traffic and response status |
| `cloudivision_http_request_duration_seconds{method,route}` | histogram | API latency |
| `cloudivision_http_errors_total{method,route,status}` | counter | API 4xx/5xx responses |
| `cloudivision_webhook_events_total{provider,outcome}` | counter | accepted, replayed, and invalid-signature webhook outcomes |
| `cloudivision_audit_write_failures_total` | counter | audit event and webhook-index persistence failures |

Metric labels are intentionally bounded. Commit SHA, repository URL, branch, actor/email, Event ID, image digest, and arbitrary error text are never labels. Project/repository labels are currently omitted. The release stuck gauge is the sole resource-identity exception: namespace/name are necessary to page on an actionable object, and series are removed when the Release becomes terminal, changes phase, or is deleted. Prometheus retention still governs historical series.

Dashboard BuildRun/Release “distribution” panels show status-update rate because the controller does not publish per-object phase gauges. Use `kubectl get ... -A` for authoritative current counts.

## Alert response

### Controller reconcile errors

Break down `cloudivision_reconcile_errors_total` by controller, inspect controller logs and Kubernetes Events for the same interval, then check API discovery, RBAC, provider availability, and invalid referenced resources. Avoid restarting until evidence is captured.

### API 5xx rate

Split requests by route and status, inspect API logs with request IDs, then check Kubernetes API connectivity and PostgreSQL/provider dependencies. Confirm readiness before restoring webhook traffic.

### Invalid webhook signatures

Identify the bounded provider label, compare delivery timestamps in the provider UI, and verify the Kubernetes Secret reference and key rotation. Treat an unexplained spike as possible probing. Never log or paste the webhook secret.

### Audit database errors

Check PostgreSQL reachability, TLS/authentication, pool exhaustion, disk capacity, and migrations. CR status remains runtime-authoritative, but audit continuity and webhook idempotency may be degraded. Restore persistence before relying on audit reports.

### Stuck BuildRun

Inspect the labeled runner Job and Pod, image pull state, quota, ServiceAccount/RBAC, NetworkPolicy, node scheduling, and the PipelineTemplate timeout. Preserve logs. Retry as a new BuildRun only after identifying whether the original Job can still complete.

### Stuck Release

Inspect Release conditions and phase, GitOps commit/PR, Argo CD or Flux health, credentials, repository availability, and the configured deployment timeout. Do not bypass GitOps with a direct cluster apply.

### Runner Job failures

Group failures by Kubernetes Events and runner log reason. Check registry/repository access and template commands before retrying. A platform-wide spike usually indicates a shared dependency rather than application code.

### GitOps push failures

Pause promotions, validate repository credentials, branch protection, rate limits, network/DNS, and remote availability. Reconciliation is designed to reuse stored commit state; verify idempotency before manually editing the GitOps repository.

Thresholds in the supplied PrometheusRule are safe starting points, not universal SLOs. Tune `for` durations and thresholds using observed load and PipelineTemplate/Release timeouts.

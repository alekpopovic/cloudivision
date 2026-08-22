# Performance and scale harness

`make scale-test` performs a safe limited generation check and does not contact a cluster. Live mode requires an explicit disposable context:

```sh
SCALE_TEST_LIVE=true SCALE_TEST_CONTEXT=kind-cloudivision SCALE_SCENARIO=100-buildruns make scale-test
```

Supported scenarios are `limited`, `100-buildruns`, `1000-buildruns` (20 projects), `50-runner-jobs`, `10-releases`, `100-webhooks`, `large-logs`, and `api-list`. The release scenario requires a writable GitOps repository and compatible provider credentials. Webhook load requires `SCALE_WEBHOOK_COMMAND`, which must generate a unique signed event using `SCALE_EVENT_NUMBER`. The 100MB log scenario is intentionally manual because it can exhaust node and log storage; use an isolated cluster and enforce quotas.

The generator labels every object with `cloudivision.io/scale-run` and the live script deletes only objects with that label. It refuses non-kind contexts unless `SCALE_TEST_ALLOW_NON_KIND=true`. Never run 1,000 builds on a shared cluster. Start with at least three worker nodes, 8 vCPU/16GiB each, registry/source mirrors, metrics-server, Prometheus, kube-state-metrics, and explicit ResourceQuota. Fifty runner Jobs can require roughly 12.5 CPU and 12.5GiB at the generated limits.

Use the Grafana dashboard to capture controller/API CPU and memory, Kubernetes request rate, reconciliation duration, queue/build/release duration, and failures. Set `SCALE_API_BASE_URL` to measure list/log latency. Browser rendering should be profiled with the Angular production build; the BuildRun UI requests at most 100 server-filtered rows. Copy `results-template.md` for each repeatable run and include warm/cold cache distinctions.

Known bottlenecks: API filtering currently lists namespace objects before applying filters/pagination; Kubernetes API list cost therefore still grows with namespace size. Runner admission has configurable controller reconciliation concurrency but no strict global/per-project active Job limiter yet. The intended design is a global token budget plus per-project weighted semaphore, persisted through Job/BuildRun state so controller restarts do not oversubscribe. PostgreSQL pool/write latency, Git provider limits, registry pulls, scheduler throughput, and log storage can dominate before controller CPU.

Controller concurrency is configured with `controller.concurrency.project`, `buildRun`, and `release` Helm values. Increase gradually while watching Kubernetes API throttling and workqueue metrics from controller-runtime. The cache indexes BuildRuns by project, repository, and webhook event ID and Releases by BuildRun reference.

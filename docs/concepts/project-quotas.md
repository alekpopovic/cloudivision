# Project quotas

Project quotas prevent one project from consuming all runner capacity:

```yaml
spec:
  quotas:
    maxConcurrentBuildRuns: 4
    maxQueuedBuildRuns: 20
    maxCPU: "2"
    maxMemory: 4Gi
    maxBuildDurationSeconds: 1800
    maxArtifactsSize: 500Mi
    maxLogSize: 50Mi
```

Before creating a Job or PipelineRun, the BuildRun controller counts executor
workloads for the same Project. When all concurrency slots are occupied, the
BuildRun remains `Queued`, has no runner workload, records a
`ConcurrencyQuotaReached` condition, and is reconsidered after a short delay.
It starts automatically when a slot becomes free. If a new BuildRun would
exceed `maxQueuedBuildRuns`, it becomes `Failed` with reason `QuotaExceeded`;
the controller emits a Kubernetes warning Event.

Zero or omitted limits are treated as unbounded. `maxBuildDurationSeconds`
overrides a looser PipelineTemplate timeout. CPU and memory maxima clamp the
runner's per-build resource limits and requests. Artifact and stored-log byte
limits are passed to the runner and enforced before persistence. Kubernetes
ResourceQuota and LimitRange remain recommended as a second isolation layer.

Metrics expose current queued BuildRuns, quota denials, and active workloads by
Project. Project names are an intentionally bounded label because Projects are
administratively created tenant boundaries; avoid generating them per build or
user request.

# 51. Project Quotas and Runner Concurrency

```text
Continue working on the cloudivision project.

Task:
Add project quotas and runner concurrency controls.

Goal:
Prevent one project or user from exhausting the cluster.

Add Project.spec.quotas:
- maxConcurrentBuildRuns int
- maxQueuedBuildRuns int
- maxCPU string optional
- maxMemory string optional
- maxBuildDurationSeconds int
- maxArtifactsSize string optional
- maxLogSize string optional

Controller behavior:
1. Before creating runner Job, count active BuildRuns for project.
2. If maxConcurrentBuildRuns is reached:
   - set BuildRun phase=Queued.
   - requeue after configured interval.
3. If queue limit exceeded:
   - fail BuildRun with reason QuotaExceeded.
4. Enforce timeout from project quota if stricter than template timeout.
5. Add metrics:
   - queued BuildRuns
   - quota denied BuildRuns
   - active BuildRuns by project if cardinality acceptable

API/UI:
- Show quota status on Project detail.
- Show queued reason on BuildRun detail.

Tests:
- max concurrency reached -> Queued
- queued BuildRun starts when slot frees
- queue limit exceeded -> Failed
- timeout limit applied
- repeated reconcile does not create extra jobs

Docs:
- docs/concepts/project-quotas.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Per-project concurrency works.
- Queued state is visible in API/UI.
- QuotaExceeded is clear and auditable.
```

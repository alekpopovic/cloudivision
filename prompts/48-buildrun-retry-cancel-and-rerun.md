# 48. BuildRun Retry, Cancel and Rerun

```text
Continue working on the cloudivision project.

Task:
Add retry, cancel and rerun support for BuildRuns.

Goal:
Users should be able to recover from transient failures and stop builds safely.

API endpoints:
- POST /api/v1/build-runs/{namespace}/{name}/cancel
- POST /api/v1/build-runs/{namespace}/{name}/retry
- POST /api/v1/build-runs/{namespace}/{name}/rerun

Behavior:
1. Cancel:
- If BuildRun is Pending/Queued/Running, delete or stop runner Job.
- Set phase=Cancelled.
- Add condition Cancelled.
- Emit event.

2. Retry:
- Only allowed for Failed/Cancelled BuildRun.
- Create a new BuildRun with same spec.
- Add annotation:
  - cloudivision.io/retry-of=<old-buildrun>
- Do not mutate old BuildRun into running again.

3. Rerun:
- Allowed for Succeeded/Failed/Cancelled.
- Create a new BuildRun with same spec.
- Add annotation:
  - cloudivision.io/rerun-of=<old-buildrun>

4. Controller:
- Terminal BuildRuns remain terminal.
- New BuildRun gets new Job.
- Cancellation is idempotent.

Angular UI:
- Add Cancel button for running builds.
- Add Retry button for failed builds.
- Add Rerun button for completed builds.
- Show relation to original BuildRun.

CLI:
- cloudivision build cancel
- cloudivision build retry
- cloudivision build rerun

Tests:
- cancel running BuildRun
- retry failed BuildRun
- rerun succeeded BuildRun
- retry running BuildRun denied
- repeated cancel is safe

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Cancel/retry/rerun work through API.
- UI exposes actions with safe confirmation.
- Original BuildRun history is preserved.
```

# Prompt 22 — Conformance Test Suite

Phase: v0.1 Hardening

```text
Continue working on the cloudivision project.

Task:
Create a conformance test suite for cloudivision.

Goal:
Add repeatable tests that prove cloudivision works as a platform on a clean Kubernetes cluster.

Location:
- /test/conformance

Create:
- /test/conformance/run.sh
- /test/conformance/lib.sh
- /test/conformance/scenarios/
- /test/conformance/fixtures/

Required scenarios:
1. Manual BuildRun success:
   - apply Project, Repository, PipelineTemplate and BuildRun fixtures
   - wait for runner Job
   - wait for BuildRun status.phase=Succeeded
   - assert startedAt/completedAt
   - assert logs are available through API if installed
2. Manual BuildRun failure:
   - create failing PipelineTemplate and BuildRun
   - wait for Failed
   - assert failure.reason and failure.message are non-empty
3. No duplicate Job:
   - create BuildRun, force multiple reconciles if possible, assert only one Job exists with cloudivision.io/buildrun label
4. Webhook idempotency:
   - create Repository with webhook enabled
   - send valid GitHub webhook payload
   - assert BuildRun is created
   - resend same payload and assert no duplicate BuildRun
5. GitOps release smoke test:
   - create successful BuildRun with gitOps.enabled=true
   - assert Release is created
   - if local test Git repo configured, assert GitOps commit; otherwise explicit skip

Script behavior:
- Use set -euo pipefail.
- Print clear progress messages.
- Fail fast on real failures.
- Use explicit timeout for every wait.
- On failure print kubectl get buildruns -A, describe buildrun, jobs, pods, relevant pod logs and controller logs if available.
- Do not silently pass skipped tests; skips must be explicit and counted.

Makefile:
- Add target conformance.

Documentation:
- Add docs/testing/conformance.md with prerequisites, local/kind execution, environment variables and troubleshooting.

Acceptance criteria:
- make conformance exists.
- Conformance suite has at least success, failure and duplicate Job scenarios.
- Scripts are executable.
- Every scenario has clear timeout behavior.
- Failures print useful debug information.
- docs/testing/conformance.md exists.
```

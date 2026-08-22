# 74. v0.2 Final Gate

```text
Continue working on the cloudivision project.

Task:
Perform the v0.2 final gate review.

Goal:
Decide whether cloudivision is ready for v0.2.0.

Run or verify:
- go test ./...
- go vet ./...
- npm ci in /web
- npm run build in /web
- npm test in /web if configured
- helm template charts/cloudivision
- make security-check
- make conformance
- make upgrade-test
- limited make scale-test if available
- dogfood BuildRun if environment exists

Review v0.2 goals:
1. Image build
- BuildKit works.
- Registry credentials work.
- Digest captured.
- Image push errors are clear.

2. Webhooks
- GitHub signature verification works.
- Event idempotency works.
- Branch/PR filters work.
- Invalid webhooks do not create BuildRuns.

3. GitOps
- helm-values strategy works.
- Release state machine is clear.
- Argo CD/Flux status optional but safe.
- Release is idempotent.

4. UI
- BuildRun debugging improved.
- Logs viewer useful.
- First-run wizard works.
- Repository onboarding works.

5. Security
- security-check passes.
- runner threat model complete.
- RBAC reviewed.
- secrets redacted.

Output:
Create docs/readiness/v0.2-final-gate.md with:
- summary
- pass/fail table
- critical blockers
- high-priority blockers
- non-blocking known issues
- commands run
- test results
- release recommendation:
  - ship
  - ship with known issues
  - do not ship

Acceptance criteria:
- v0.2 final gate document exists.
- It contains clear release recommendation.
- Safe critical fixes are implemented.
- Remaining risks are documented.
```

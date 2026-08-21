# Prompt 40 — v0.1 Final Gate: Hardening Sprint Review

Phase: Release

```text
Continue working on the cloudivision project.

Task:
Perform the final v0.1 hardening sprint review.

Goal:
Decide whether cloudivision is ready for v0.1.0 or what exactly blocks it.

Run or verify:
- go test ./...
- go vet ./...
- npm ci in /web
- npm run build in /web
- npm test in /web if configured
- helm template charts/cloudivision
- make security-check
- make conformance
- make upgrade-test if available
- make scale-test in limited mode if available

Review:
1. Fresh install: clean kind install, Helm install, API reachable, Angular UI reachable, controller healthy.
2. First user path: quickstart, create Project/Repository/PipelineTemplate, trigger BuildRun, view logs, troubleshoot failure.
3. Security: no privileged default, no docker.sock, no hostPath default, runner RBAC reviewed, webhook verification tested, secrets redacted.
4. Reliability: BuildRun and Release controllers idempotent, status conflicts handled, runner failures clear, logs accessible.
5. Documentation: README, quickstart, Helm docs, troubleshooting and security limitations are accurate.
6. Release readiness: v0.1.0 changelog draft, known issues, artifacts and upgrade limitations documented.

Output:
Create docs/readiness/v0.1-final-gate.md with:
- Summary
- Pass/fail table
- Critical blockers
- High-priority blockers
- Non-blocking known issues
- Commands run
- Test results
- Recommended v0.1.0 decision: ship, ship with known issues, or do not ship

Then:
- Fix safe critical issues discovered during review.
- Do not add major new features.
- Do not hide failing tests.
- Be explicit about unresolved items.

Acceptance criteria:
- docs/readiness/v0.1-final-gate.md exists.
- It contains a clear release recommendation.
- All commands run are listed with results.
- Safe critical fixes are implemented.
- Remaining risks are documented.
```

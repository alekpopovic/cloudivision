# Prompt 41 — v0.2 Roadmap Planning

Phase: Roadmap

```text
Continue working on the cloudivision project.

Task:
Create the v0.2 roadmap after v0.1 hardening.

Goal:
Define the next product increment after the first usable release.

Create:
- docs/roadmap/v0.2.md
- docs/roadmap/backlog.md

v0.2 should focus on:
1. Image build and registry reliability:
   - BuildKit adapter hardening
   - registry auth
   - digest capture
   - image push retry
   - immutable image usage
2. GitHub webhook production readiness:
   - signature verification
   - event idempotency
   - branch filters
   - PR/push event distinction
   - replay protection
3. GitOps Release MVP:
   - direct commit mode
   - helm-values strategy
   - release state machine
   - Argo CD sync/health reader
   - deployment timeout
4. Angular CI/CD UX:
   - BuildRun debugging improvements
   - Repository onboarding wizard
   - Release detail page
   - provider health page
5. Security baseline:
   - runner threat model
   - security-check in CI
   - least-privilege RBAC review
   - NetworkPolicy examples

Backlog categories:
- v0.2 must-have
- v0.2 nice-to-have
- v0.3 candidates
- future enterprise features

For each roadmap item include:
- problem
- proposed solution
- acceptance criteria
- risks
- dependencies

Acceptance criteria:
- docs/roadmap/v0.2.md exists.
- docs/roadmap/backlog.md exists.
- Roadmap is realistic and not a feature dump.
- v0.2 priorities are tied to real user value.
```

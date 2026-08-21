# Prompt 28 — Release State Machine Hardening

Phase: Release / GitOps

```text
Continue working on the cloudivision project.

Task:
Harden the Release lifecycle and GitOps state machine.

Goal:
Replace vague Release states with precise states that help users understand where deployment failed or is waiting.

Desired phases:
- Pending
- AwaitingApproval
- PreparingGitOpsChange
- GitOpsChangeCommitted
- WaitingForSync
- Deployed
- RolledBack
- FailedValidation
- FailedApproval
- FailedGitClone
- FailedGitCommit
- FailedGitPush
- FailedProviderStatus
- TimedOut

Rules:
1. AwaitingApproval: approval required and not granted means no clone/push. Surface approver and timestamp when present.
2. PreparingGitOpsChange: validating image, environment and GitOps config before clone/update/commit.
3. GitOpsChangeCommitted: set after commit/push, store gitCommit and safe GitOps metadata.
4. WaitingForSync: after GitOps commit when waiting for Argo CD/Flux/provider status.
5. Deployed: only when provider confirms deployment or generic provider policy says commit is enough. For Argo CD, keep syncStatus and healthStatus separate.
6. Failure states: use specific phase/reason, preserve message and emit Kubernetes Event.
7. Idempotency: no duplicate commits for same Release; use annotations/status gitCommit to detect applied change.
8. Timeout: add configurable deployment timeout and set TimedOut when exceeded.

Tests:
- approval required blocks GitOps update
- approval allows GitOps update
- Git clone failure -> FailedGitClone
- Git commit failure -> FailedGitCommit
- Git push failure -> FailedGitPush
- successful commit -> GitOpsChangeCommitted
- provider synced/healthy -> Deployed
- repeated reconcile does not create duplicate commits
- timeout -> TimedOut

Docs:
Update docs/concepts/release.md with phase diagram, state descriptions and troubleshooting by phase.

Acceptance criteria:
- go test ./... passes.
- Release lifecycle is documented.
- Failure phases are specific.
- GitOps updates are idempotent.
- Argo CD syncStatus and healthStatus are represented separately.
```

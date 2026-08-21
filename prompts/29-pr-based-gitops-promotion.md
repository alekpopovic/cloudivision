# Prompt 29 — PR-Based GitOps Promotion

Phase: Release / GitOps

```text
Continue working on the cloudivision project.

Task:
Add a design and initial implementation for PR-based GitOps promotion.

Goal:
Production releases should optionally create a pull request instead of directly pushing to the GitOps branch.

Add Release strategy options:
- direct-commit
- pull-request

CRD/API changes:
Extend Release or Environment GitOps policy with:
- promotionMode: direct-commit | pull-request
- pullRequest: titleTemplate optional, bodyTemplate optional, targetBranch, reviewers optional, labels optional

Provider abstraction:
Create interface PullRequestProvider with CreateOrUpdatePullRequest and ReadPullRequestStatus.

Support providers:
- GitHub skeleton
- GitLab skeleton
- Generic unsupported provider with clear error

Behavior:
1. direct-commit keeps existing GitOps behavior.
2. pull-request creates branch from target branch, commits GitOps image update, opens PR/MR, sets Release phase, stores PR URL/reference.
3. Reconcile idempotent: no duplicate PRs; update existing PR when needed.
4. Merged PR moves Release to WaitingForSync.
5. Closed unmerged PR moves to FailedApproval or FailedPromotion.

Angular UI:
- Show PR link, promotion mode and merge status on Release detail.
- Do not implement full PR review UI.

Docs:
- docs/gitops/pr-based-promotion.md

Tests:
- direct-commit still works
- pull-request mode creates one PR
- repeated reconcile does not duplicate PR
- closed unmerged PR blocks release
- merged PR allows sync wait

Acceptance criteria:
- go test ./... passes.
- PR-based promotion is optional.
- Existing direct-commit flow is not broken.
- Provider skeleton returns clear unsupported errors when not configured.
- Release UI shows PR metadata when present.
```

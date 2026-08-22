# 57. Release Promotion and Rollback

```text
Continue working on the cloudivision project.

Task:
Implement release promotion and rollback.

Goal:
Users should promote a built image across environments and rollback failed releases.

API:
- POST /api/v1/releases/{namespace}/{name}/promote
- POST /api/v1/releases/{namespace}/{name}/rollback

Promote request:
{
  "targetEnvironmentRef": "staging",
  "actor": "..."
}

Rollback request:
{
  "targetReleaseRef": "previous-release",
  "actor": "...",
  "reason": "..."
}

Behavior:
1. Promotion:
- Create new Release targeting next environment.
- Reuse image digest from source Release/BuildRun.
- Apply approval policy of target environment.
- Record audit event.

2. Rollback:
- Create new Release using image from previous successful Release.
- Mark relationship:
  - rollbackOf
  - rollbackTo
- Do not mutate old Release.
- GitOps update follows normal Release flow.
- Record audit event.

3. UI:
- Add Promote action.
- Add Rollback action.
- Show release lineage.

4. Policy:
- Production promotion requires approval by default.
- Rollback to unsigned/no-digest image can be blocked by policy.

Tests:
- promote dev to staging
- promote staging to production awaiting approval
- rollback to previous release
- rollback blocked by policy
- release lineage shown

Docs:
- docs/concepts/release-promotion.md
- docs/concepts/rollback.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Promotion creates a new Release.
- Rollback creates a new Release.
- Existing Release history remains immutable.
```

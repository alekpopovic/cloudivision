# 54. GitOps Helm Values Production Flow

```text
Continue working on the cloudivision project.

Task:
Harden the helm-values GitOps update strategy.

Goal:
Make the primary GitOps release path production-ready.

Behavior:
1. Clone GitOps repo.
2. Read configured values file.
3. Update:
   - image.repository
   - image.tag
   - image.digest if present
4. Preserve YAML formatting where practical.
5. Commit change with clear message.
6. Push to branch.
7. Store gitCommit in Release status.
8. Ensure repeated reconcile does not create duplicate commits.
9. Detect if desired image is already present.
10. Add clear error states for clone/parse/update/commit/push.

Config:
- gitOps.path
- valuesFile
- imageRepositoryField default image.repository
- imageTagField default image.tag
- imageDigestField default image.digest

Tests:
- update simple values.yaml
- update nested values.yaml
- missing file
- invalid YAML
- already updated no-op
- push failure
- repeated reconcile no duplicate commit

Docs:
- docs/gitops/helm-values.md

Acceptance criteria:
- go test ./... passes.
- helm-values strategy is idempotent.
- Failure reasons are specific.
- Release status records git commit.
- Docs include a complete example.
```

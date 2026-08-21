# Prompt 08 — GitOps Integration and Release Controller

Phase: Release / GitOps

```text
Continue working on the cloudivision project.

Task:
Implement GitOps integration and the Release controller.

Goal:
When a BuildRun succeeds and has spec.gitOps.enabled=true, create a Release CR. Release controller updates the GitOps repository. It must not deploy application manifests directly to the target namespace.

Part A: BuildRun -> Release
1. Extend BuildRun controller so that when BuildRun transitions to Succeeded and spec.gitOps.enabled=true, it creates a Release if one does not exist.
2. Use a stable Release name such as <buildrun-name>-<environment> or stable hash.
3. Release spec includes projectRef, buildRunRef, environmentRef if available, image repository/tag/digest and strategy=gitops.

Part B: Release controller
1. Load Release.
2. If approval.required=true and approvedBy is empty, set phase=AwaitingApproval and do not update GitOps repo.
3. If ready to deploy:
   - clone GitOps repo
   - update file according to strategy: helm-values, kustomize-image or raw-yaml
   - commit with message "cloudivision: release <release-name> image <image>"
   - push to target branch
   - write gitCommit to status
   - set phase=Deploying
4. If Environment connects to Argo CD Application, read Application CR if CRD exists, write syncStatus/healthStatus and set Deployed when Synced and Healthy.
5. If Argo CD CRD does not exist, do not fail; keep Deploying and add ArgoCDStatusUnavailable condition.

Implementation:
- Create /internal/gitops Provider interface with UpdateImage and ReadDeploymentStatus.
- Implement GitRepositoryProvider.
- Implement ArgoCDStatusReader using unstructured Kubernetes object.
- Add clear errors.

Acceptance criteria:
- go test ./... passes.
- BuildRun success creates Release only once.
- Release controller is idempotent.
- GitOps update functions have tests using a temporary git repo.
- No direct kubectl apply to application namespace.
- Argo CD integration is optional.
```

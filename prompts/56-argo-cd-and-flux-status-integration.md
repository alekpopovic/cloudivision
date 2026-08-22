# 56. Argo CD and Flux Status Integration

```text
Continue working on the cloudivision project.

Task:
Improve Argo CD and Flux status integration.

Goal:
Release status should reflect actual GitOps provider sync and health state.

Argo CD:
- Read Application CR as unstructured object.
- Extract:
  - status.sync.status
  - status.health.status
  - status.operationState.phase if present
  - status.reconciledAt if present
- Map to Release.status.deployment.

Flux:
- Add skeleton/readers for:
  - Kustomization
  - HelmRelease
- Extract Ready condition if present.
- Extract observed revision if present.

Behavior:
1. If provider CRD is missing, set condition ProviderUnavailable, not fatal.
2. If app/resource is missing, set condition DeploymentResourceMissing.
3. If synced and healthy, Release can become Deployed.
4. If degraded, set failure/warning condition depending on policy.
5. Keep syncStatus and healthStatus separate.

Angular UI:
- Show provider.
- Show sync status.
- Show health status.
- Show last observed revision/time.

Tests:
- Argo CD Application synced/healthy
- Argo CD out-of-sync
- Argo CD degraded
- Argo CD CRD missing
- Flux resource ready
- Flux resource missing

Docs:
- docs/gitops/argocd.md
- docs/gitops/flux.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Provider status is optional.
- Missing provider does not crash controller.
- Release UI shows separate sync and health.
```

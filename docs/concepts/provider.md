# Provider

Providers are small adapters for Git, registry, GitOps, secrets, builds, notifications, and supply-chain tools. The registry exposes bounded capabilities and non-mutating health checks through the API/UI.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Environment
metadata: {name: staging, namespace: ci}
spec:
  projectRef: storefront
  displayName: Staging
  namespace: storefront-staging
  type: staging
  requiresApproval: false
  gitOps: {provider: argocd, applicationName: storefront-staging, namespace: argocd}
```

`GET /api/v1/providers` lists capability metadata and `/health` checks compiled adapters. “Healthy” means the adapter is available, not that every external repository/application credential is valid. See the deeper [provider registry note](providers.md) and [adding a provider](../development/adding-provider.md).

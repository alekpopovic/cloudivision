# Environment

An Environment describes a GitOps deployment target and its approval/artifact policy. It does not grant credentials or deploy directly.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Environment
metadata: {name: production, namespace: ci}
spec:
  projectRef: storefront
  displayName: Production
  namespace: storefront-production
  type: production
  requiresApproval: true
  gitOps: {provider: argocd, applicationName: storefront, namespace: argocd}
  policy:
    requireImageDigest: true
    requireSignedImages: true
    requireSBOM: true
    blockCriticalVulnerabilities: true
```

Production requires approval by schema and policy. Provider namespace/application fields identify status to read; target-cluster authorization remains the GitOps controller's responsibility.

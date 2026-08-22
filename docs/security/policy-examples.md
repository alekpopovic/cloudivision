# Policy examples

Require immutable, signed, scanned images with an SBOM in production:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Environment
metadata:
  name: production
  namespace: team-a
spec:
  projectRef: storefront
  displayName: Production
  namespace: storefront-production
  type: production
  requiresApproval: true
  gitOps:
    provider: argocd
    applicationName: storefront
    namespace: argocd
  policy:
    requireImageDigest: true
    requireSignedImages: true
    requireSBOM: true
    blockCriticalVulnerabilities: true
```

A rejected release exposes actionable status:

```yaml
status:
  phase: FailedValidation
  policy:
    allowed: false
    reason: PolicyDenied
    message: this environment requires an immutable image digest
    violations:
      - policy: ImageMustHaveDigest
        severity: error
        message: this environment requires an immutable image digest
        fieldPath: spec.image.digest
  conditions:
    - type: PolicyDenied
      status: "True"
      reason: PolicyDenied
```

The default runner policy rejects `spec.security.allowPrivileged: true`. It also detects Docker socket and hostPath requests in pipeline step arguments. Store required credentials in Kubernetes Secrets and reference them through supported secret fields; never pass secret values in params or command arguments.

When the API denies a trigger it returns:

```json
{
  "code": "policy_denied",
  "message": "caller is not authorized to trigger builds",
  "violations": [
    {
      "policy": "CanTriggerBuild",
      "severity": "error",
      "message": "caller is not authorized to trigger builds",
      "fieldPath": "spec.triggeredBy"
    }
  ]
}
```

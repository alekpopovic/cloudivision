# Policy engine

cloudivision evaluates business and security rules through the code-based evaluator in `internal/policy`. The evaluator is deliberately independent from Kubernetes clients: callers load the relevant CRs, pass them as policy input, and receive a deterministic decision with a reason and zero or more violations.

Policy evaluation happens at three boundaries:

- the API checks authorization policy before creating a manually triggered `BuildRun` and returns HTTP 403 with `code: policy_denied` and structured `violations`;
- the BuildRun controller evaluates the referenced project, repository, and pipeline template before creating an executor workload;
- the Release controller evaluates the artifact and target Environment before changing a GitOps repository.

Controller decisions are stored under `status.policy`. A denial also creates a `PolicyDenied` condition and warning Event. The Angular BuildRun and Release detail pages render the decision, policy name, severity, message, and field path.

## Default rules

The default policy forbids privileged runners, Docker socket use, hostPath use, and unknown repository, build, or GitOps providers. It validates project boundaries and production approvals. Environment policy fields opt into immutable digests, signatures, SBOMs, and blocking critical vulnerabilities.

Policies are currently compiled into the controller and API binaries. ConfigMap policy and external engines such as OPA are intentionally outside this version; keeping a small `Evaluator` interface allows a later implementation without coupling domain logic to an external engine.

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
  policy: {requireImageDigest: true, requireSignedImages: true, requireSBOM: true, blockCriticalVulnerabilities: true}
```

## Decision shape

Each decision includes `allowed`, `reason`, `message`, `evaluatedAt`, and `violations`. A violation identifies the stable policy name, severity, human-readable message, and the field path operators should correct. Clients should key automation on `reason` and `policy`, not message text.

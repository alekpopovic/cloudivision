# Project

A Project is the tenancy and defaults boundary for related repositories, pipelines, builds, and releases. The controller can create/label its namespace and installs namespaced runner ServiceAccount/RBAC and optional NetworkPolicy.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Project
metadata: {name: storefront, namespace: ci}
spec:
  displayName: Storefront
  ownerTeam: commerce
  namespace: storefront-ci
  defaultRegistry: ghcr.io/acme
  defaultBranch: main
  serviceAccountName: cloudivision-runner
  isolation: {createNamespace: true, podSecurityLevel: restricted, networkPolicyMode: defaultDeny}
```

Project separation is not a complete multi-tenant security boundary by itself. Use cluster admission, namespace quotas, NetworkPolicies, scoped credentials, and OIDC/RBAC. A BuildRun must reference resources from the same project and namespace.

# Prompt 13 — Helm Chart and Cluster Installation

Phase: Installation

```text
Continue working on the cloudivision project.

Task:
Create a Helm chart for installing cloudivision into a Kubernetes cluster.

Chart path:
- /charts/cloudivision

The chart should install:
- CRDs
- controller Deployment
- API Deployment
- web Deployment
- Services
- ServiceAccounts
- RBAC
- ConfigMap
- Secret template placeholder
- optional Ingress
- optional PostgreSQL dependency or external database config
- optional NetworkPolicy
- optional Pod Security labels for namespace

values.yaml should support:
- global.imageRegistry
- controller.image.repository/tag/pullPolicy
- api.image.repository/tag/pullPolicy
- runner.image.repository/tag/pullPolicy
- web.image.repository/tag/pullPolicy
- api.auth.mode
- api.defaultNamespace
- api.cors.allowedOrigins
- database.enabled
- database.externalUrlSecret
- audit.backend
- ingress.enabled
- ingress.className
- ingress.hosts
- resources for controller/api/web
- securityContext/podSecurityContext
- networkPolicy.enabled
- tekton.enabled
- argocd.enabled
- web.config.apiBaseUrl

Steps:
1. Add Chart.yaml.
2. Add values.yaml with secure defaults.
3. Add templates for all components.
4. Add NOTES.txt with quickstart commands.
5. Add helm template test to Makefile target helm-template.
6. Add docs/install/helm.md with install instructions, runner/API/web config, ingress, audit backend, GitOps/Argo CD options and Angular runtime config.

Security:
- Default must not be cluster-admin except where installation/controller CRD operations require cluster-level permissions.
- Clearly separate install-time and runtime RBAC.
- Pods should use non-root securityContext where possible.
- Do not enable privileged by default.
- Do not mount docker.sock.

Acceptance criteria:
- helm template charts/cloudivision passes.
- Rendered YAML has no privileged=true.
- Services and deployments use recommended Kubernetes labels.
- Values are documented.
- Installation is namespace-aware.
```

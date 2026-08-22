# 64. Helm Chart Production Hardening

```text
Continue working on the cloudivision project.

Task:
Harden the Helm chart for production-style installs.

Improve values:
- image pull secrets
- pod annotations
- nodeSelector
- tolerations
- affinity
- topologySpreadConstraints
- priorityClassName
- serviceMonitor optional
- ingress TLS
- extraEnv
- extraVolumes only with explicit security warning
- resource defaults

Security:
- PodSecurityContext non-root
- seccomp RuntimeDefault
- readOnlyRootFilesystem where practical
- no privileged
- no hostPath default
- no docker.sock
- minimal RBAC
- NetworkPolicy optional but documented

Operations:
- liveness/readiness probes
- startup probe if needed
- PDB optional
- HPA optional for API/web
- controller leader election config
- metrics service

Tests:
- helm template default
- helm template with ingress
- helm template with networkPolicy
- helm template with database enabled
- security-check on rendered chart

Docs:
- docs/reference/helm-values.md
- docs/operations/install-helm.md

Acceptance criteria:
- helm template passes.
- make security-check passes.
- values.yaml is documented.
- Chart supports production customization without unsafe defaults.
```

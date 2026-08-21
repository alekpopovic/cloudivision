# Prompt 10 — Multi-Tenancy, RBAC and Pod Security Hardening

Phase: Security

```text
Continue working on the cloudivision project.

Task:
Add multi-tenancy and security hardening for cloudivision.

Goal:
Each Project can have an isolated namespace, ServiceAccount and minimal RBAC for build runners.

Part A: Project controller
1. If Project.spec.isolation.createNamespace=true, create namespace Project.spec.namespace and add labels app.kubernetes.io/managed-by=cloudivision and cloudivision.io/project=<project-name>.
2. If podSecurityLevel=baseline or restricted, add Pod Security Admission labels enforce/audit/warn.
3. Create runner ServiceAccount from Project.spec.serviceAccountName or cloudivision-runner.
4. Create namespaced Role/RoleBinding with only necessary permissions: get/list/watch/update/patch BuildRun where practical, get Repository/PipelineTemplate, create/update Events. No cluster-admin.
5. If networkPolicyMode=defaultDeny, create default deny NetworkPolicy.
6. If networkPolicyMode=egressAllowList, create skeleton or document extension.

Part B: Build Job security
1. JobExecutor uses Project runner ServiceAccount.
2. Pod spec uses automountServiceAccountToken only when runner must update BuildRun status.
3. Pod/container securityContext: runAsNonRoot, seccomp RuntimeDefault where possible, allowPrivilegeEscalation false, privileged false, drop ALL capabilities.
4. Reject PipelineTemplate.spec.security.allowPrivileged=true unless global config explicitly allows.
5. Add resource defaults and default activeDeadlineSeconds.

Part C: Secret redaction
1. Add central redaction package.
2. Mask token, password, secret, authorization, private key and client secret.
3. Apply redaction in API logs, runner logs and webhook logs.

Acceptance criteria:
- go test ./... passes.
- Project controller creates namespace, ServiceAccount, Role and RoleBinding.
- RBAC is namespaced wherever possible.
- Job does not use privileged, hostPath or docker.sock.
- Pod Security labels are added when requested.
- Secret redaction tests cover at least 10 cases.
```

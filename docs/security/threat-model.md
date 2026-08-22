# Threat model

Assets include Kubernetes/API credentials, source and GitOps repositories, registry artifacts, signing keys, webhook secrets, audit records, and deployment approvals. Trust boundaries exist at public ingress, identity provider, Kubernetes API, project namespaces, untrusted repository code, external CLIs/providers, PostgreSQL, registry, and GitOps controller.

Primary threats are forged/replayed webhooks, malicious build scripts, credential exfiltration, cross-project access, privileged workload escape, dependency/tool compromise, mutable image substitution, GitOps tampering, approval spoofing, secret/log leakage, denial of service, and stale provider status.

```yaml
spec:
  isolation:
    createNamespace: true
    podSecurityLevel: restricted
    networkPolicyMode: defaultDeny
```

Controls include signature verification/idempotency, OIDC/RBAC, namespaced runner RBAC, unprivileged Jobs, policy evaluation, resource/deadline limits, redaction, immutable artifact policy, GitOps-only CD, audit events, and security render tests.

Residual limitations: repository code still runs with project runner credentials/network; namespace boundaries depend on cluster admission/runtime; no complete tenant network policy generator; external provider and registry compromise remain; supply-chain evidence durability/signing is incomplete; v1alpha1 upgrades need operator review. See the detailed [runner threat model](runner-threat-model.md).

# Prompt 27 — Runner Threat Model and Security Conformance

Phase: Security

```text
Continue working on the cloudivision project.

Task:
Create a runner threat model and enforce a security baseline for runner workloads.

Goal:
The runner executes untrusted repository code. Its security posture must be explicit, tested and documented.

Deliverables:
1. docs/security/runner-threat-model.md
2. test/security/no-privileged.sh
3. test/security/no-docker-sock.sh
4. test/security/no-hostpath.sh
5. test/security/rbac-minimal.sh
6. Makefile target: security-check

Threat model document should include:
Assets:
- Kubernetes API credentials
- Git credentials
- registry credentials
- GitOps repository credentials
- build artifacts
- logs
- image digests
- release metadata
- namespace resources

Threats:
- malicious repository code
- malicious dependency install script
- secret exfiltration
- lateral movement in cluster
- privilege escalation through privileged containers
- Docker socket escape
- hostPath escape
- poisoning image tags
- overwriting GitOps repository
- leaking secrets through logs
- abusing runner ServiceAccount

Controls:
- no docker.sock mount
- no privileged containers by default
- no hostPath by default
- non-root runner
- dropped capabilities
- seccomp RuntimeDefault where possible
- least-privilege ServiceAccount
- namespace isolation
- NetworkPolicy where enabled
- resource limits
- timeouts
- secret redaction
- immutable image digests for releases
- approval gates for production
- audit events

Security scripts:
- Render Helm chart.
- Inspect rendered YAML.
- Fail if privileged=true exists.
- Fail if docker.sock is mounted.
- Fail if hostPath is used without explicit allowlist.
- Fail if runner has cluster-admin.
- Fail if runner Role can list all Secrets.
- Print clear violation messages.

Acceptance criteria:
- docs/security/runner-threat-model.md exists.
- make security-check exists.
- security-check fails on privileged/docker.sock/hostPath/cluster-admin violations.
- Current Helm output passes security-check.
- go test ./... passes.
```

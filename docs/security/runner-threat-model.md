# Runner threat model

The cloudivision runner executes repository-controlled commands and dependency
installation scripts. Repository contents are untrusted even when the repository
belongs to the organization. This document defines the default security boundary;
exceptions require a documented risk acceptance and a narrowly scoped chart option.

## Assets and trust boundaries

The assets at risk are Kubernetes API credentials, Git and registry credentials,
GitOps repository credentials, build artifacts, logs, image digests, release
metadata, and all namespace resources. The runner Pod is inside the project
namespace but outside the platform control-plane trust boundary. The controller,
API, registry, source provider, GitOps repository, and dependency registries are
separate trust boundaries. Data received from any of them must be treated as
untrusted until authenticated and validated.

## Threats

| Threat | Example impact | Required controls |
| --- | --- | --- |
| Malicious repository code or dependency install script | Reads credentials, mines cryptocurrency, or attacks the API | non-root execution, dropped capabilities, resource limits, timeout, namespace isolation |
| Secret exfiltration or log leakage | Credentials leave the cluster in network requests or build output | least-privilege ServiceAccount, scoped secret projection, secret redaction, optional NetworkPolicy, audit events |
| Lateral movement | Runner reads or mutates another workload | per-project namespace and ServiceAccount, namespaced RBAC, NetworkPolicy where enabled |
| Privilege escalation | Code obtains host or cluster privileges | no privileged containers, `allowPrivilegeEscalation: false`, all capabilities dropped, RuntimeDefault seccomp |
| Docker socket or hostPath escape | Code controls the node runtime or reads host files | no `docker.sock`, no `hostPath`, rootless image builder |
| Image tag poisoning | A mutable tag deploys different content after approval | capture and promote immutable image digests |
| GitOps overwrite | Runner rewrites unrelated desired state or branch history | GitOps credentials stay outside runner Jobs, narrow repository/path policy, approval gates for production |
| Runner ServiceAccount abuse | Code lists Secrets or creates arbitrary workloads | minimal namespaced Role, no cluster-admin, no broad Secret listing |

## Enforced baseline

Runner workloads run as non-root, use RuntimeDefault seccomp, cannot become
privileged, and drop all Linux capabilities. Docker socket and hostPath mounts are
forbidden by default. Each build has CPU/memory requests and limits and an active
deadline. A project runner uses its own namespaced ServiceAccount and Role. Network
isolation is installed when the project enables it; production clusters should also
enforce Kubernetes Pod Security Standards and egress policy at admission time.

Credentials must be projected only into the step that needs them, must never be
printed, and must pass through log redaction. GitOps write credentials belong to
the Release controller, not a build runner. Releases retain immutable image
digests, require configured production approvals, and emit Kubernetes Events and
audit records for security-relevant state transitions.

## Conformance and residual risk

Run `make security-check` before publishing a chart. It renders the chart and fails
on `privileged: true`, Docker socket references, non-allowlisted hostPath volumes,
cluster-admin references, or RBAC that can enumerate all Secrets. The scripts are
deliberately independent so a rendered manifest can be checked directly:

```sh
helm template cloudivision charts/cloudivision --include-crds > /tmp/rendered.yaml
test/security/no-privileged.sh /tmp/rendered.yaml
test/security/no-docker-sock.sh /tmp/rendered.yaml
test/security/no-hostpath.sh /tmp/rendered.yaml
test/security/rbac-minimal.sh /tmp/rendered.yaml
```

Static checks do not prove runtime isolation. Cluster admission policy, node
hardening, network enforcement, registry immutability, credential rotation, and
monitoring remain operator responsibilities. Any allowlist change must include a
specific path, owner, justification, expiry, and regression test.

# 67. Multi-Cluster Architecture

```text
Continue working on the cloudivision project.

Task:
Design and start implementing multi-cluster support.

Goal:
Allow cloudivision to manage builds/deployments across multiple Kubernetes clusters in future versions.

Create concept:
- Cluster resource or Provider config

Possible CRD:
ClusterTarget
Spec:
- displayName
- kubeconfigSecretRef
- context
- type: build, deploy, both
- defaultNamespace
- labels
Status:
- phase
- version
- reachable
- conditions

Behavior:
1. For now, local cluster remains default.
2. External clusters are optional.
3. API can list cluster targets.
4. Controller can health check cluster target.
5. Do not run builds on external cluster until secure model is defined.

Docs:
- docs/architecture/multi-cluster.md
- docs/security/multi-cluster.md

Tests:
- local cluster provider
- missing kubeconfig
- unreachable cluster
- health status

Acceptance criteria:
- go test ./... passes.
- Multi-cluster design is documented.
- Local single-cluster behavior is not broken.
- External cluster support is clearly marked experimental.
```

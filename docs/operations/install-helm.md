# Install with Helm

Render and inspect first, then install:

```sh
helm lint charts/cloudivision
helm template cloudivision charts/cloudivision --namespace cloudivision --include-crds > /tmp/cloudivision.yaml
helm upgrade --install cloudivision charts/cloudivision --namespace cloudivision --create-namespace
kubectl -n cloudivision rollout status deployment --all --timeout=5m
curl --fail http://localhost:8080/readyz # after API port-forward
```

Production installations should set immutable component tags, OIDC, TLS ingress, explicit CORS, resource limits, audit storage, NetworkPolicy, and external Secret management in a reviewed values file:

```sh
helm upgrade --install cloudivision charts/cloudivision -n cloudivision -f production-values.yaml
```

A production values file commonly includes:

```yaml
controller:
  replicaCount: 2
  leaderElection: true
  podDisruptionBudget:
    enabled: true
api:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 6
web:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 6
ingress:
  enabled: true
  className: nginx
  tls:
    - secretName: cloudivision-tls
      hosts: [cloudivision.example.com]
networkPolicy:
  # Enable only after adding the cluster-specific ingress/egress rules below.
  enabled: false
serviceMonitor:
  enabled: true
imagePullSecrets:
  - name: registry-credentials
```

Before enabling NetworkPolicy, add environment-specific egress rules for DNS, the Kubernetes API, Git/registry/provider endpoints, PostgreSQL, and monitoring. A policy rendered by this chart cannot infer cluster service CIDRs. `serviceMonitor.enabled` requires the Prometheus Operator CRDs; API/web autoscaling requires the Kubernetes metrics API.

All containers run non-root with `RuntimeDefault` seccomp, dropped capabilities, no privilege escalation, and a read-only root filesystem by default. The chart provides writable `emptyDir` runtime paths. Treat `extraVolumes` and `extraVolumeMounts` as a security exception: never add hostPath, Docker/container-runtime sockets, devices, or broadly scoped Secrets. Run these gates on every values change:

```sh
make helm-test
make security-check
helm template cloudivision charts/cloudivision -n cloudivision -f production-values.yaml > /tmp/production.yaml
```

CRDs in `crds/` install only on initial Helm install and remain after uninstall. Follow the [upgrade runbook](upgrade.md) for schema changes. Auth-disabled mode is development-only.

## Troubleshooting

Use `helm get values/manifest`, `kubectl get events`, and controller/API logs. If CR kinds are unknown, apply the CRD bundle and wait for `Established`. If images fail, verify `global.imageRegistry`, tags, pull Secrets, architecture, node scheduling constraints, and egress. Probe failures with a read-only filesystem usually indicate a custom volume or image expecting an undeclared writable path. More examples are in [troubleshooting](troubleshooting.md) and the deeper [Helm note](../install/helm.md).

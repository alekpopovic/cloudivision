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

CRDs in `crds/` install only on initial Helm install and remain after uninstall. Follow the [upgrade runbook](upgrade.md) for schema changes. Auth-disabled mode is development-only.

## Troubleshooting

Use `helm get values/manifest`, `kubectl get events`, and controller/API logs. If CR kinds are unknown, apply the CRD bundle and wait for `Established`. If images fail, verify `global.imageRegistry`, tags, pull Secrets, architecture, and egress. More examples are in [troubleshooting](troubleshooting.md) and the deeper [Helm note](../install/helm.md).

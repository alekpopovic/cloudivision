# Helm values reference

This page is maintained with [`charts/cloudivision/values.yaml`](../../charts/cloudivision/values.yaml).
When adding or renaming a value, update both files in the same commit.

| Value | Default | Meaning |
| --- | --- | --- |
| `global.imageRegistry` | `ghcr.io/alekpopovic/cloudivision` | Registry prepended to component repositories |
| `namespace.create` | `false` | Create the release namespace |
| `namespace.podSecurity.enabled` | `true` | Add Pod Security labels |
| `namespace.podSecurity.level` | `restricted` | Enforced Pod Security level |
| `controller.replicaCount` | `1` | Controller replicas |
| `controller.concurrency.project` | `2` | Concurrent Project reconciles |
| `controller.concurrency.buildRun` | `10` | Concurrent BuildRun reconciles |
| `controller.concurrency.release` | `5` | Concurrent Release reconciles |
| `controller.image.{repository,tag,pullPolicy}` | `controller`, `dev`, `IfNotPresent` | Controller image |
| `controller.metricsBindAddress` | `:8080` | Metrics listener |
| `controller.healthProbeBindAddress` | `:8081` | Probe listener |
| `controller.leaderElection` | `false` | Controller leader election |
| `controller.resources` | requests `100m/128Mi`, limits `500m/512Mi` | Controller resources |
| `api.replicaCount` | `1` | API replicas |
| `api.image.{repository,tag,pullPolicy}` | `api`, `dev`, `IfNotPresent` | API image |
| `api.service.{type,port}` | `ClusterIP`, `8080` | API service |
| `api.auth.mode` | `disabled` | `disabled` (development) or `oidc` |
| `api.auth.oidc.{issuerUrl,clientId,audience,jwksUrl}` | empty | OIDC validation settings |
| `api.auth.groupMappings` | `[]` | Group-to-role/scope mappings |
| `api.metrics.enabled` | `true` | API metrics |
| `api.defaultNamespace` | `default` | API namespace fallback |
| `api.cors.allowedOrigins` | localhost ports 4200 and 4201 | Allowed browser origins |
| `api.resources` | requests `100m/128Mi`, limits `500m/512Mi` | API resources |
| `runner.image.{repository,tag,pullPolicy}` | `runner`, `dev`, `IfNotPresent` | Build Job runner image |
| `web.replicaCount` | `1` | Web replicas |
| `web.image.{repository,tag,pullPolicy}` | `web`, `dev`, `IfNotPresent` | Web image |
| `web.service.{type,port}` | `ClusterIP`, `80` | Web service |
| `web.config.apiBaseUrl` | empty | Runtime API URL; empty means same origin |
| `web.resources` | requests `50m/64Mi`, limits `250m/256Mi` | Web resources |
| `database.enabled` | `false` | Enable PostgreSQL audit URL injection |
| `database.externalUrlSecret.{name,key}` | empty, `database-url` | Existing database URL Secret |
| `audit.backend` | `log` | `noop`, `log`, or `postgres` |
| `ingress.enabled` | `false` | Create ingress |
| `ingress.className`, `ingress.annotations`, `ingress.hosts`, `ingress.tls` | empty/sample host | Ingress routing and TLS |
| `networkPolicy.enabled` | `false` | Create NetworkPolicies |
| `tekton.enabled` | `false` | Enable the Tekton executor |
| `argocd.enabled` | `false` | Enable Argo CD read integration |
| `podSecurityContext` | non-root, RuntimeDefault seccomp | Pod-level defaults |
| `securityContext` | no escalation, non-privileged, drop all capabilities | Container defaults |
| `imagePullSecrets` | `[]` | Registry pull Secrets |
| `nameOverride`, `fullnameOverride` | empty | Resource name overrides |

Inspect the exact rendered result before installation:

```sh
helm show values charts/cloudivision
helm template cloudivision charts/cloudivision --namespace cloudivision \
  --include-crds -f my-values.yaml
```

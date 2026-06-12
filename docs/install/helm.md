# Helm install

Install cloudivision into a namespace:

```sh
helm install cloudivision charts/cloudivision --namespace cloudivision --create-namespace
```

Render locally, including CRDs:

```sh
helm template cloudivision charts/cloudivision --include-crds
```

## Runner image

The controller passes `CLOU_DIVISION_RUNNER_IMAGE` into BuildRun Jobs from:

```yaml
runner:
  image:
    repository: runner
    tag: dev
```

Set `global.imageRegistry` or fully adjust each component image repository/tag.

## API and web

The API defaults to development auth mode:

```yaml
api:
  auth:
    mode: disabled
  defaultNamespace: default
  cors:
    allowedOrigins:
      - http://localhost:4200
```

The Angular UI reads runtime config from `/assets/config.json`. Set:

```yaml
web:
  config:
    apiBaseUrl: https://cloudivision.example.com
```

## Ingress

Ingress is disabled by default. Enable it with:

```yaml
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: cloudivision.example.com
      paths:
        - path: /
          pathType: Prefix
          service: web
        - path: /api
          pathType: Prefix
          service: api
```

## Audit backend and database

PostgreSQL is optional. Runtime state for BuildRuns and Releases remains in Kubernetes CRD status.
The chart is configured for external database URLs by default and does not vendor a PostgreSQL subchart
yet.

Use log audit by default:

```yaml
audit:
  backend: log
database:
  enabled: false
```

Use an external database secret:

```yaml
audit:
  backend: postgres
database:
  externalUrlSecret:
    name: cloudivision-database
    key: database-url
```

Apply `/internal/audit/migrations` before setting `audit.backend=postgres`.

## GitOps and Argo CD

Argo CD status reading is optional:

```yaml
argocd:
  enabled: true
```

Tekton support is disabled by default and must be explicitly enabled:

```yaml
tekton:
  enabled: true
```

## Security defaults

The chart separates install/controller permissions from runtime API/web permissions. The controller uses
a ClusterRole because it watches cloudivision CRDs and may create project namespaces. API and web are
namespaced. Pods run non-root where possible, drop all capabilities, and do not mount `docker.sock`.

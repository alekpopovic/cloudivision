# Configuration reference

Helm values render the component environment. Prefer chart values in Kubernetes;
use environment variables for direct local execution.

## Controller

| Variable | Purpose |
| --- | --- |
| `CLOUDIVISION_METRICS_BIND_ADDRESS` | Metrics listen address |
| `CLOUDIVISION_HEALTH_PROBE_BIND_ADDRESS` | Health listen address |
| `CLOUDIVISION_LEADER_ELECTION` | Enable leader election |
| `CLOUDIVISION_PROJECT_CONCURRENCY` | Project worker count |
| `CLOUDIVISION_BUILDRUN_CONCURRENCY` | BuildRun worker count |
| `CLOUDIVISION_RELEASE_CONCURRENCY` | Release worker count |
| `CLOU_DIVISION_RUNNER_IMAGE` | Runner image used by Jobs |
| `CLOU_DIVISION_ENABLE_TEKTON` | Enable optional Tekton execution |
| `CLOU_DIVISION_ALLOW_PRIVILEGED_BUILDS` | Exceptional privileged-build opt-in; avoid in production |

## API

| Variable | Purpose |
| --- | --- |
| `CLOU_DIVISION_DEFAULT_NAMESPACE` | Default CR namespace |
| `CLOU_DIVISION_AUTH_MODE` | `disabled` (development only) or `oidc` |
| `CLOU_DIVISION_OIDC_ISSUER_URL` | Trusted OIDC issuer |
| `CLOU_DIVISION_OIDC_CLIENT_ID` | OIDC client ID |
| `CLOU_DIVISION_OIDC_AUDIENCE` | Required token audience |
| `CLOU_DIVISION_OIDC_JWKS_URL` | Optional explicit JWKS endpoint |
| `CLOU_DIVISION_AUTH_GROUPS_FILE` | Group mapping YAML file |
| `CLOU_DIVISION_CORS_ALLOWED_ORIGINS` | Comma-separated exact browser origins |
| `CLOU_DIVISION_METRICS_ENABLED` | Enable API metrics |
| `CLOU_DIVISION_AUDIT_BACKEND` | `noop`, `log`, or `postgres` |
| `CLOU_DIVISION_DATABASE_URL` | PostgreSQL URL required by the postgres backend |
| `KUBECONFIG` | Kubernetes client configuration for direct execution |

## CLI

`CLOU_DIVISION_API_URL`, `CLOU_DIVISION_TOKEN`, and
`CLOU_DIVISION_NAMESPACE` provide CLI defaults. Flags override environment, which
overrides the CLI YAML config. See the [CLI reference](cli.md).

Never put bearer tokens, database passwords, webhook secrets, or Git credentials
in ConfigMaps, command history, checked-in values, or logs. Use Kubernetes Secrets
and a production secret manager.

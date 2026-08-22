# Provider registry

cloudivision exposes integrations through a central, thread-safe registry keyed by
`type/name`. The registry makes capabilities discoverable and prevents controllers,
the API, and the UI from relying on scattered string switches.

Provider categories are Git, registry, GitOps, secrets, build, notifications, and
supply-chain. The initial catalog contains:

- generic Git plus GitHub and GitLab skeletons
- a generic OCI registry descriptor
- Kubernetes Secret projection
- generic GitOps and Argo CD status
- BuildKit
- noop notifications
- noop, Syft, Grype, and Cosign supply-chain adapters

`GET /api/v1/providers` returns stable provider metadata and capabilities.
`GET /api/v1/providers/health` executes all health checks concurrently and returns
health, message, and `checkedAt`. The Angular Providers page refreshes this health
view every 30 seconds.

Health describes adapter availability, not permission to every external object.
For example, Argo CD is registered and healthy while per-Application Kubernetes
RBAC is still checked during Release reconciliation. CLI-backed adapters report
unhealthy when their binary is absent. Noop providers are healthy by design.

Duplicate registration and unknown lookups return typed, clear errors. They do not
replace an existing provider or panic. Startup treats duplicate catalog entries as
a configuration error.

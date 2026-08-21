# Prompt 32 — Provider Adapter Registry

Phase: Providers

```text
Continue working on the cloudivision project.

Task:
Create a formal provider adapter registry for cloudivision.

Goal:
Avoid hardcoding integrations throughout the codebase. Providers should be discoverable, testable and configurable.

Provider categories:
- Git providers
- Registry providers
- GitOps providers
- Secret providers
- Build providers
- Notification providers
- Supply-chain providers

Create package structure:
- /internal/provider
- /internal/provider/git
- /internal/provider/registry
- /internal/provider/gitops
- /internal/provider/secrets
- /internal/provider/build
- /internal/provider/notifications
- /internal/provider/supplychain

Core interfaces:

type Provider interface {
    Name() string
    Type() string
    HealthCheck(ctx context.Context) ProviderHealth
    Capabilities() []Capability
}

type ProviderHealth struct {
    Healthy bool
    Message string
    CheckedAt time.Time
}

type Capability struct {
    Name string
    Description string
}

Provider registry:
- Register(provider Provider)
- Get(type, name string)
- List()
- HealthCheckAll(ctx)

Initial providers:
- generic Git provider
- GitHub provider skeleton
- GitLab provider skeleton
- Kubernetes Secrets provider
- generic GitOps provider
- Argo CD status provider
- BuildKit build provider
- noop notification provider
- noop supply-chain providers

API:
- GET /api/v1/providers
- GET /api/v1/providers/health

Angular UI:
Add Providers page with type, capabilities, health, message and last checked.

Docs:
- docs/concepts/providers.md
- docs/development/adding-provider.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Provider registry exists.
- Existing provider code is migrated where reasonable.
- API exposes provider list and health.
- UI shows provider health.
- Unsupported providers return clear errors rather than panics.
```

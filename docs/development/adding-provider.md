# Adding a provider

1. Choose the narrow category under `internal/provider/`. Do not put domain logic
   in the registry package; adapt an existing Git, build, GitOps, or supply-chain
   interface.
2. Implement `provider.Provider`: stable lowercase `Name` and `Type`, bounded
   `HealthCheck`, and user-facing capabilities.
3. Register the constructor in `configureProviderRegistry` in `cmd/api/main.go`.
4. Add unit tests for capability metadata, healthy/unhealthy behavior, missing
   configuration, and unsupported operations.
5. Document credentials, least-privilege permissions, network requirements, and
   whether the adapter is production-ready or a skeleton.

```go
type Provider interface {
    Name() string
    Type() string
    HealthCheck(ctx context.Context) ProviderHealth
    Capabilities() []Capability
}
```

Health checks must honor context cancellation, avoid logging secrets, and avoid
mutating external state. Keep them cheap: the API and UI may invoke them regularly.
Use the generic `provider.Static` descriptor only when static or binary-presence
health is accurate. Providers that call remote APIs should implement the interface
directly and return a safe message without tokens, URLs containing credentials, or
raw response bodies.

Registration is explicit so the compiled catalog is reviewable. A future dynamic
plugin system can build on the same interface, but provider registration does not
grant Kubernetes RBAC or external credentials by itself.

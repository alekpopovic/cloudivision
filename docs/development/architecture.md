# Architecture

cloudivision is a Kubernetes-native CI/CD control plane. Desired state and durable
runtime state live in CRDs; reconcilers turn that state into Jobs or optional
Tekton PipelineRuns. CI produces immutable artifacts, while CD changes a GitOps
repository and leaves rollout to Argo CD or Flux.

```text
Git webhook / CLI / UI
          |
       Go API ---- audit backend
          |
    Kubernetes CRDs
          |
 controller manager ---- Job or Tekton executor ---- runner
          |
     Release controller ---- GitOps repository ---- Argo CD / Flux
```

The API exposes DTOs instead of CRD structs. Provider and executor interfaces keep
Git, registry, build, supply-chain and GitOps implementations outside domain
state transitions. PostgreSQL is optional and is used for audit records, not as
the source of truth for builds or releases.

## Repository map

- `api/v1alpha1`: Kubernetes types and validation.
- `cmd/{controller,api,runner,cloudivision}`: binaries.
- `internal/controller`: idempotent reconcilers.
- `internal/{executor,runner,build,gitops,provider}`: execution boundaries.
- `internal/api`: HTTP handlers and DTOs.
- `web`: Angular application.
- `charts/cloudivision`: installation chart.

## Development checks

```sh
gofmt -w ./api ./cmd ./internal
go test ./...
go vet ./...
npm --prefix web ci
npm --prefix web run build
npm --prefix web test -- --watch=false --browsers=ChromeHeadless
helm template cloudivision charts/cloudivision --include-crds >/tmp/cloudivision.yaml
```

See [controllers](controllers.md), [runner](runner.md), [API](api.md), and the
[Kubernetes-native architecture decision](../adr/0001-kubernetes-native-architecture.md).

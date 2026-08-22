# 44. Registry Provider and Credentials

```text
Continue working on the cloudivision project.

Task:
Implement registry provider and credential handling.

Goal:
Allow BuildRuns to push images to registries safely.

Provider types:
- generic docker registry
- GHCR
- GitLab Registry
- Harbor
- ECR skeleton
- GCR/Artifact Registry skeleton
- ACR skeleton

Create:
- /internal/provider/registry

Core interface:
type RegistryProvider interface {
    Name() string
    Login(ctx context.Context, req LoginRequest) (*LoginResult, error)
    ResolveImage(ctx context.Context, req ImageRequest) (*ImageRef, error)
    ReadDigest(ctx context.Context, req ImageRequest) (string, error)
    HealthCheck(ctx context.Context) ProviderHealth
}

Credentials:
- Kubernetes Secret reference
- username/password
- token
- dockerconfigjson
- cloud provider skeletons for future

Security:
- Never log credential values.
- Redact registry tokens.
- Mount or inject credentials only into runner job.
- Do not give runner access to all secrets in namespace.
- Document required secret formats.

CRD/API:
Extend Project or Repository if needed:
- registry:
  - provider
  - imagePrefix
  - credentialSecretRef

Docs:
- docs/registry/providers.md
- docs/registry/credentials.md

Tests:
- credential parsing
- redaction
- unsupported provider
- missing secret
- dockerconfigjson handling

Acceptance criteria:
- go test ./... passes.
- Generic registry credentials work.
- Registry provider health can be reported.
- Runner only receives required registry credentials.
- Missing credentials produce clear BuildRun failure reason.
```

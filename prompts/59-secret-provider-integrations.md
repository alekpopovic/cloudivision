# 59. Secret Provider Integrations

```text
Continue working on the cloudivision project.

Task:
Improve secret provider integration.

Goal:
Support Kubernetes Secrets safely now and prepare for External Secrets/Vault later.

Providers:
- Kubernetes Secrets
- External Secrets skeleton
- Vault skeleton

Create:
- /internal/provider/secrets

Interface:
type SecretProvider interface {
    Resolve(ctx context.Context, ref SecretRef) (*ResolvedSecret, error)
    HealthCheck(ctx context.Context) ProviderHealth
}

Behavior:
1. Kubernetes Secret provider:
- resolve named secret in allowed namespace
- support selected keys only
- never return all secrets unless explicitly requested and safe
- redact values in logs

2. External Secrets skeleton:
- detect ExternalSecret CRD if installed
- report health/capability

3. Vault skeleton:
- config structure only
- return unsupported unless configured

Security:
- Runner should receive only secrets required by its BuildRun.
- BuildRun status must never contain secret values.
- API must never return secret values.
- UI can show secret presence/status only.

Docs:
- docs/security/secrets.md
- docs/operations/external-secrets.md

Tests:
- resolve existing secret key
- missing secret
- missing key
- forbidden namespace
- redaction
- API does not expose secret values

Acceptance criteria:
- go test ./... passes.
- Secret access is explicit and minimal.
- Existing registry/git credentials use secret provider where practical.
- Docs explain secret formats.
```

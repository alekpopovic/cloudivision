# Registry providers

Registry providers resolve image names, prepare scoped Docker authentication,
read manifest digests, and report health. BuildKit remains responsible for the
actual image build and push.

Configure a provider per Project:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Project
metadata:
  name: payments
  namespace: payments-ci
spec:
  displayName: Payments
  ownerTeam: platform
  namespace: payments-ci
  defaultRegistry: ghcr.io/example
  isolation:
    createNamespace: false
    podSecurityLevel: restricted
    networkPolicyMode: defaultDeny
  registry:
    provider: ghcr
    imagePrefix: ghcr.io/example
    credentialSecretRef:
      name: payments-registry
```

`imagePrefix` is prepended when a BuildRun image repository is relative. An
already fully qualified repository is preserved. BuildRun status records the
resolved repository, tag, and digest separately.

## Provider matrix

| Provider | Value | v0.2 status | Notes |
|---|---|---|---|
| OCI/Docker Registry | `generic` | implemented | Any Registry HTTP API v2 compatible service. |
| GitHub Container Registry | `ghcr` | implemented | Defaults the registry host to `ghcr.io`. |
| GitLab Container Registry | `gitlab` | implemented | Defaults the host to `registry.gitlab.com`. |
| Harbor | `harbor` | implemented | Configure the Harbor host in `imagePrefix`. |
| Amazon ECR | `ecr` | skeleton | Cloud credential exchange is not implemented. |
| Google Artifact Registry/GCR | `gcr` | skeleton | Cloud credential exchange is not implemented. |
| Azure Container Registry | `acr` | skeleton | Cloud credential exchange is not implemented. |

Selecting a cloud skeleton returns `RegistryProviderUnsupported`; it never falls
back to an unsafe or ambient cluster credential. Generic credentials may be used
with a cloud registry only when the operator deliberately supplies a compatible
short-lived token.

## Provider contract

The `internal/provider/registry.RegistryProvider` interface provides:

- `Login` to validate credentials and return Docker config bytes;
- `ResolveImage` to combine provider defaults, prefix, repository, tag, and
  digest;
- `ReadDigest` to perform an authenticated Registry v2 manifest `HEAD` and
  validate `Docker-Content-Digest`; and
- `HealthCheck` to report implemented versus skeleton state through the existing
  provider health API.

Provider health describes adapter availability, not the validity of a Project's
credential. Credential checks happen at BuildRun creation/use time so one
project's secret cannot affect global health or leak configuration.

## Failure reasons

| Reason | Meaning |
|---|---|
| `RegistryProviderUnsupported` | The provider name is unknown or a cloud skeleton was selected. |
| `RegistryImageInvalid` | The resolved repository, tag, digest, or host is invalid. |
| `RegistryCredentialsMissing` | The reference, Secret, or supported credential keys are missing. |
| `RegistryCredentialsInvalid` | The Secret format or Docker config is invalid. |

See [registry credentials](credentials.md) for Secret formats and security
boundaries, and [BuildKit builds](../build/buildkit.md) for push/digest behavior.

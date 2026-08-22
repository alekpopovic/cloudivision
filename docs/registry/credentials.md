# Registry credentials

Registry credentials are stored only in a dedicated namespaced Kubernetes
Secret. CRDs contain the Secret name and optional key, never a username,
password, token, or Docker config value.

The BuildRun controller reads only the exact referenced Secret to validate that
the Job can start. It cannot list Secrets. The Job mounts only that Secret as a
read-only volume at `/var/run/secrets/cloudivision-registry`; the runner service
account receives no Secret read permission. The runner creates a temporary
`DOCKER_CONFIG/config.json` with mode `0600`, passes only its directory path to
BuildKit, and removes it when the run finishes.

Use a dedicated Secret containing only registry authentication data. When
`credentialSecretRef.key` is set, only that one key is projected and interpreted
as Docker config JSON. When it is omitted, the supported fixed key names below
are mounted from the dedicated Secret.

## Username and password

```sh
kubectl -n payments-ci create secret generic payments-registry \
  --from-literal=username='robot-user' \
  --from-literal=password='replace-me'
```

Both keys are required for this format. Use a robot/service identity with push
permission only for the configured image prefix.

## Token

```sh
kubectl -n payments-ci create secret generic payments-registry \
  --from-literal=username='robot-user' \
  --from-literal=token='replace-me'
```

`username` is recommended and required by registries such as GHCR. A token-only
Secret is encoded with the provider's generic `oauth2` username; use it only when
the registry documents that behavior.

## Docker config JSON

The standard Kubernetes Docker config Secret works directly:

```sh
kubectl -n payments-ci create secret docker-registry payments-registry \
  --docker-server=ghcr.io \
  --docker-username='robot-user' \
  --docker-password='replace-me'
```

This produces a `.dockerconfigjson` key. A generic Secret may instead contain a
`config.json` key. The document must be valid JSON with a non-empty `auths`
object.

To use a non-standard key and project only that key:

```yaml
registry:
  provider: generic
  imagePrefix: registry.example.com/payments
  credentialSecretRef:
    name: payments-registry
    key: docker-config
```

The `docker-config` value must contain Docker config JSON.

## Rotation and troubleshooting

Secret values are read for each new runner Job, so rotate by updating the Secret
and rerunning the BuildRun. Existing Pods retain the projected value according to
Kubernetes volume update behavior; do not depend on mid-build rotation.

- `RegistryCredentialsMissing` means the Project reference, Secret, or supported
  keys are absent.
- `RegistryCredentialsInvalid` means username/password pairing or Docker config
  validation failed.
- `ImageBuildFailed` after successful credential validation may mean wrong scope,
  expired token, registry egress denial, TLS trust, or repository permission.

Credential values are registered with the runner redactor and must not appear in
BuildRun status, Kubernetes Events, audit data, or logs. Never put registry
passwords/tokens in build args, labels, image tags, parameters, or repository
URLs. Do not grant the runner `get`, `list`, or `watch` on Secrets.

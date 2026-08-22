# Image tagging and immutable releases

cloudivision generates a safe image tag when a `BuildRun` does not provide one.
The default template is `{{ .Branch }}-{{ .ShortSHA }}`. The generated value is
written to both `spec.image.tag` and `status.image.tag`, so execution and audit
views use the same resolved tag.

## Tag templates

Set a project-wide template with `spec.imageTagPolicy.defaultTagTemplate`:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Project
metadata:
  name: payments
spec:
  displayName: Payments
  ownerTeam: platform
  namespace: payments-ci
  defaultRegistry: ghcr.io/acme
  imageTagPolicy:
    defaultTagTemplate: "{{ .BuildRunName }}"
```

Templates may use `.Branch`, `.CommitSHA`, `.ShortSHA`, `.BuildRunName`, and
`.Timestamp`. Common strategies are:

- `{{ .CommitSHA }}` for a full source revision tag.
- `{{ .Branch }}-{{ .ShortSHA }}` for human-readable branch builds.
- `{{ .BuildRunName }}` for a direct Kubernetes audit trail.
- `{{ .Timestamp }}-{{ .ShortSHA }}` for time-ordered builds.

An empty branch becomes `detached`, an empty commit becomes `unknown`, and an
empty timestamp uses the Unix epoch. Values are lower-cased, unsafe characters
are replaced with `-`, and tags are limited to 128 characters. A manually
provided safe semantic-version tag such as `v2.4.1` is preserved.

## Digest-first releases

Tags are convenient labels, but they are mutable. BuildKit records the pushed
manifest digest in `BuildRun.status.image.digest`. When a digest is available,
generated `Release` resources and GitOps updates use `repository@digest` and
remove mutable tag fields from Helm values or Kustomize image overrides.

Production environments reject `latest` by default and may require a digest:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Environment
metadata:
  name: production
spec:
  projectRef: payments
  displayName: Production
  namespace: payments-production
  type: production
  requiresApproval: true
  policy:
    allowLatest: false
    requireImageDigest: true
```

Set `allowLatest: true` only for an explicit exception. For production, the
recommended policy is to keep it false and require a digest so the deployed
artifact cannot change after approval.

# Artifacts

Pipeline steps can publish test reports, binaries, SBOMs, and provenance files
after all steps complete:

```yaml
steps:
  - name: test
    image: node:24-alpine
    command: [sh, -c]
    args: ["npm test -- --reporter=junit"]
    artifacts:
      paths: ["reports/*.xml"]
      optional: false
      retentionDays: 14
```

Paths are relative to the checked-out source workspace and may use glob
patterns. Directories are expanded recursively. Required patterns fail the
BuildRun when they match no regular files; optional patterns are skipped.
Individual files are capped at 100 MiB. The runner calculates a SHA-256 digest,
uploads the bytes, then writes immutable metadata to
`BuildRun.status.artifacts` (`name`, `path`, media `type`, `size`, `digest`,
backend `ref`, and `createdAt`).

Artifact paths must remain inside the workspace. Absolute paths, parent
traversal, escaping symlinks, `.git`, `.env`, and secret/password/private-key/
credential-like components are rejected. This denylist is defense in depth:
pipelines should place publishable output in a dedicated `reports/` or `dist/`
directory and must never write credentials there.

Storage is disabled by default, so pipelines without artifact storage retain
their existing behavior. Local development can enable durable files with:

```yaml
artifacts:
  backend: local
  local:
    root: /var/lib/cloudivision/artifacts
    existingClaim: cloudivision-artifacts
```

In Kubernetes, the PVC must be accessible in the namespace where both the API
and runner execute; cross-namespace projects need a storage backend designed
for that topology. Object storage and OCI artifact adapters are currently
interface skeletons and must not be selected for production.

List metadata with
`GET /api/v1/build-runs/{namespace}/{name}/artifacts` and download an item with
`GET /api/v1/build-runs/{namespace}/{name}/artifacts/{artifactName}`. The
BuildRun detail page exposes the same list and download action.

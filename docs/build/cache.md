# Runner workspace and dependency cache

Dependency caching is disabled by default. Enable it explicitly on an individual
`PipelineTemplate` when repeated dependency downloads justify the additional
trust and storage cost.

```yaml
spec:
  cache:
    enabled: true
    mode: pvc
    key: npm-lock-v1
    paths:
      - node_modules
      - .npm
    restoreKeys:
      - npm-main
    ttlSeconds: 604800
```

The runner clones into a fresh `/workspace/source`, restores cached paths after
checkout and before pipeline steps, and publishes a replacement snapshot only
after every step, image build, and supply-chain hook succeeds. Missing entries
are cache misses, not build failures. Unsafe absolute paths, `..` traversal and
symbolic links are rejected. `cache.local.maxSize` limits the bytes saved by a
runner (default `2Gi`).

## Modes

- `pvc` stores path snapshots on the PVC named by
  `cache.local.existingClaim`. The claim must support the access pattern needed
  by the API pod and concurrent runner Jobs; RWX storage is recommended.
- `registry` configures the existing BuildKit registry cache. The cache tag is
  derived from the project, repository, and cache key, so unrelated projects do
  not share a cache tag. It requires an enabled BuildKit image build and registry
  credentials with cache push/pull access.
- `object-storage` is an explicit interface skeleton in this release. Selecting
  it returns a bounded `CacheRestoreFailed` until an object-store adapter is
  configured in a later prompt.

Every PVC path and registry reference includes cryptographic hashes of both the
project and Repository CR names before the user-provided key. The user key alone
can therefore never cause cross-project sharing. `restoreKeys` are searched only
inside that same scope.

## Purging

Purge all entries for a project:

```sh
curl -fsS -X POST \
  'http://localhost:8080/api/v1/projects/payments/cache/purge?namespace=ci'
```

Add `repository=checkout` to purge only one repository. The API validates that
the Repository belongs to the Project. The Angular Project detail page exposes
the project-wide purge action.

## Security warning

Caches contain untrusted build output. A compromised branch or dependency hook
can poison a key and affect later builds in the same project/repository scope.
Use keys that change with lockfiles, keep production and untrusted pull-request
pipelines on distinct Projects or cache keys, use short TTLs, and purge after a
suspected compromise. Never cache credentials, signing keys, Docker config, or
other secret-bearing paths. Cache hits are an optimization and must not be
treated as proof that dependencies are authentic.

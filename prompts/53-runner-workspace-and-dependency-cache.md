# 53. Runner Workspace and Dependency Cache

```text
Continue working on the cloudivision project.

Task:
Add optional runner workspace and dependency cache support.

Goal:
Speed up builds while keeping cache security explicit.

Cache types:
- disabled default
- PVC-based cache
- registry cache for BuildKit
- object storage skeleton

PipelineTemplate.spec.cache:
- enabled bool
- mode enum: pvc, registry, object-storage
- key string optional
- paths []string optional
- restoreKeys []string optional
- ttlSeconds int optional

Security:
- Cache disabled by default.
- Cache must be scoped by project/repository.
- Do not share cache across unrelated projects by default.
- Document cache poisoning risks.
- Allow cache purge.

Runner:
1. Restore cache before steps.
2. Run steps.
3. Save cache after successful build or configurable behavior.
4. Respect max cache size if configured.

API:
- POST /api/v1/projects/{name}/cache/purge
- optional cache status endpoint

Angular UI:
- Show cache enabled/disabled on PipelineTemplate detail.
- Add purge cache action if API exists.

Docs:
- docs/build/cache.md
- include security warning

Tests:
- cache disabled default
- cache key generation
- project isolation
- purge behavior skeleton

Acceptance criteria:
- go test ./... passes.
- Cache is opt-in.
- Cache cannot accidentally share across projects by default.
- Docs explain risks and configuration.
```

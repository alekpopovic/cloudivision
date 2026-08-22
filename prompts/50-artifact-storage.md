# 50. Artifact Storage

```text
Continue working on the cloudivision project.

Task:
Add artifact storage support.

Goal:
BuildRuns should be able to publish files such as test reports, binaries, SBOMs and provenance documents.

Backends:
- local filesystem for dev
- object storage skeleton
- OCI artifact skeleton

Create:
- /internal/artifacts

Interface:
type ArtifactStore interface {
    Put(ctx context.Context, req PutArtifactRequest) (*ArtifactRef, error)
    Get(ctx context.Context, ref ArtifactRef) (*Artifact, error)
    List(ctx context.Context, req ListArtifactsRequest) ([]ArtifactRef, error)
    Delete(ctx context.Context, ref ArtifactRef) error
}

CRD:
Extend PipelineTemplate steps or build config with:
- artifacts:
  - paths []string
  - optional bool
  - retentionDays int optional

Extend BuildRun.status:
- artifacts:
  - name
  - path
  - type
  - size
  - digest
  - ref
  - createdAt

Runner:
1. After steps complete, collect configured artifact paths.
2. Calculate digest.
3. Upload to artifact store.
4. Update BuildRun.status.artifacts.
5. Do not upload secrets accidentally; document exclusions.

API:
- GET /api/v1/build-runs/{namespace}/{name}/artifacts
- GET /api/v1/build-runs/{namespace}/{name}/artifacts/{artifactName}

Angular UI:
- Add Artifacts tab to BuildRun detail.

Docs:
- docs/concepts/artifacts.md

Acceptance criteria:
- go test ./... passes.
- Artifacts can be collected from demo pipeline.
- Artifact metadata appears in BuildRun status.
- UI shows artifact list.
- Default works without external object storage by using disabled/noop mode.
```

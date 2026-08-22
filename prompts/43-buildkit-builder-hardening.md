# 43. BuildKit Builder Hardening

```text
Continue working on the cloudivision project.

Task:
Harden the BuildKit image builder adapter.

Goal:
Make image build reliable enough for v0.2.

Improve:
- BuildKit availability checks
- buildctl command construction
- rootless mode documentation
- context directory handling
- Dockerfile path handling
- build args
- target stage
- platforms
- labels
- push behavior
- digest capture
- clear errors

Extend PipelineTemplate.spec.build if needed:
- buildArgs map[string]string
- target string optional
- platforms []string optional
- labels map[string]string optional
- cache:
  - enabled bool
  - mode enum: inline, registry, local
  - ref string optional

Runner behavior:
1. Validate contextDir exists.
2. Validate Dockerfile exists.
3. Build image.
4. Push image if push=true.
5. Capture digest.
6. Update BuildRun.status.image.repository/tag/digest.
7. Add conditions:
   - ImageBuildStarted
   - ImageBuilt
   - ImagePushed
   - ImageDigestCaptured

Tests:
- build command generation
- missing Dockerfile
- missing buildctl
- push disabled
- digest parsing
- build failure maps to clear BuildRun failure reason

Docs:
- docs/build/buildkit.md

Acceptance criteria:
- go test ./... passes.
- BuildKit adapter returns clear errors.
- Runner updates image digest when available.
- BuildKit remains optional and does not break non-build pipelines.
- No Docker socket or privileged default is introduced.
```

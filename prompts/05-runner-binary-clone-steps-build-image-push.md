# Prompt 05 — Runner Binary: Clone, Steps, Build Image, Push

Phase: Build Execution

```text
Continue working on the cloudivision project.

Task:
Implement /cmd/runner as the real build runner for the Kubernetes Job executor.

MVP runner behavior:
1. Read env vars BUILD_RUN_NAME, BUILD_RUN_NAMESPACE, PROJECT_NAME, REPOSITORY_URL, REVISION, BRANCH, PIPELINE_TEMPLATE_NAME, IMAGE_REPOSITORY, IMAGE_TAG, GITOPS_ENABLED.
2. Initialize Kubernetes in-cluster client.
3. Load BuildRun, Repository and PipelineTemplate.
4. Clone the repository into /workspace/source.
5. Checkout commitSHA/revision if present, otherwise branch.
6. Execute pipeline steps in order.
7. If PipelineTemplate.spec.build.enabled=true, build image using a build adapter. Default adapter: BuildKit. Do not use docker.sock. Do not assume privileged mode.
8. Push image if build.push=true.
9. Update BuildRun conditions: RepositoryCloned, StepsCompleted, ImageBuilt, ImagePushed.
10. On failure: mark BuildRun Failed, write reason/message and do not log secrets.
11. On success: mark BuildRun Succeeded and write image tag/digest if available.

Implementation:
- Add /internal/git client for clone/checkout.
- Add /internal/executor/steps for command execution.
- Add /internal/build interface:

  type ImageBuilder interface {
      Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
  }

  BuildRequest fields: ContextDir, Dockerfile, ImageRepository, ImageTag, Push, Env.
  BuildResult fields: ImageRepository, Tag, Digest, SBOMPath optional.

- Implement BuildKitBuilder that invokes buildctl or buildctl-daemonless.sh if available.
- If buildctl is unavailable, return a clear setup error.
- Use os/exec with context timeout.
- Add log redaction helper that masks known secret env values.

Acceptance criteria:
- go test ./... passes.
- Runner can execute a pipeline without image build for a simple repo.
- Runner has tests for env parsing, redaction, phase/condition updates and command failure behavior.
- No Docker socket mount assumptions.
- No secret values in logs.
- Build adapter is an interface, not hardcoded into domain logic.
```

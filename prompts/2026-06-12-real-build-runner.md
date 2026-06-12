# Real build runner

User prompt:

```text
Continue working on the cloudivision project.

Task:
Implement /cmd/runner as the real build runner for the Kubernetes Job executor.

MVP runner behavior:
1. Read env vars:
   - BUILD_RUN_NAME
   - BUILD_RUN_NAMESPACE
   - PROJECT_NAME
   - REPOSITORY_URL
   - REVISION
   - BRANCH
   - PIPELINE_TEMPLATE_NAME
   - IMAGE_REPOSITORY
   - IMAGE_TAG
   - GITOPS_ENABLED
2. Initialize Kubernetes in-cluster client.
3. Load BuildRun, Repository and PipelineTemplate.
4. Clone the repository into /workspace/source.
5. Checkout:
   - commitSHA/revision if present
   - otherwise branch
6. Execute pipeline steps in order.
7. If PipelineTemplate.spec.build.enabled=true:
   - build the image using a build adapter
   - default adapter: BuildKit
   - do not use docker.sock
   - do not assume privileged mode
8. Push image if build.push=true.
9. Write status progress to BuildRun conditions:
   - RepositoryCloned
   - StepsCompleted
   - ImageBuilt
   - ImagePushed
10. On failure:
   - mark BuildRun as Failed
   - write reason and message
   - do not log secret values
11. On success:
   - mark BuildRun as Succeeded
   - write image tag/digest if available

Implementation:
- Add /internal/git client for clone/checkout.
- Add /internal/executor/steps for command execution.
- Add /internal/build interface:

type ImageBuilder interface {
    Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
}

BuildRequest:
- ContextDir
- Dockerfile
- ImageRepository
- ImageTag
- Push bool
- Env map[string]string

BuildResult:
- ImageRepository
- Tag
- Digest
- SBOMPath optional

- Implement BuildKitBuilder that invokes buildctl or buildctl-daemonless.sh if available.
- If buildctl is not available, return a clear error with setup guidance.
- Use os/exec with context timeout.
- Add log redaction helper that masks known secret env values.

Acceptance criteria:
- go test ./... passes.
- Runner can execute a pipeline without image build for a simple repo.
- Runner has tests for:
  - env parsing
  - redaction
  - phase/condition updates
  - command failure behavior
- No Docker socket mount assumptions.
- No secret values in logs.
- Build adapter is an interface, not hardcoded into domain logic.
```

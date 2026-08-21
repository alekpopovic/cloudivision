# Prompt 09 — Executor Interface and Tekton Adapter

Phase: Executor

```text
Continue working on the cloudivision project.

Task:
Introduce an executor abstraction and add a Tekton executor adapter without breaking the existing Kubernetes Job MVP.

Current state:
BuildRun controller directly creates Kubernetes Jobs.

Desired interface:

type PipelineExecutor interface {
    EnsureRun(ctx context.Context, req EnsureRunRequest) (*RunRef, error)
    ReadRunStatus(ctx context.Context, ref RunRef) (*RunStatus, error)
    CancelRun(ctx context.Context, ref RunRef) error
}

Implementations:
- JobExecutor
- TektonExecutor

Steps:
1. Move Job creation logic from BuildRun controller into /internal/executor/job.
2. BuildRun controller chooses executor based on BuildRun.spec.executor: job default, tekton explicit only.
3. Add /internal/executor/tekton.
4. TektonExecutor creates PipelineRun as unstructured object or typed client if dependency already exists.
5. Map BuildRun metadata/labels, Repository URL, revision, PipelineTemplate steps and image build params to PipelineRun.
6. If Tekton CRD is not installed, return clear error and set BuildRun Failed with reason TektonUnavailable.
7. Add feature flag CLOU_DIVISION_ENABLE_TEKTON=true; reject Tekton executor if not enabled.
8. JobExecutor remains default and all existing tests pass.

Acceptance criteria:
- go test ./... passes.
- JobExecutor behavior remains unchanged.
- TektonExecutor does not introduce mandatory runtime dependency.
- BuildRun controller no longer contains low-level Job/PipelineRun creation details.
- Tests exist for executor selection.
```

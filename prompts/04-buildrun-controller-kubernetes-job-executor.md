# Prompt 04 — BuildRun Controller: Kubernetes Job Executor

Phase: Build Execution

```text
Continue working on the cloudivision project.

Task:
Implement the BuildRun reconciler that creates and tracks a Kubernetes Job for every BuildRun with executor=job.

Reconcile behavior:
1. Load the BuildRun.
2. If deleting, run cleanup/finalizer logic if needed.
3. Add finalizer cloudivision.io/buildrun-finalizer if missing.
4. Validate Project, Repository and PipelineTemplate references.
5. If BuildRun is terminal (Succeeded, Failed, Cancelled), do not create a new Job.
6. If no Job exists:
   - create a Kubernetes Job in the Project namespace or BuildRun namespace
   - set ownerReference to BuildRun
   - add labels app.kubernetes.io/name=cloudivision, app.kubernetes.io/component=runner, cloudivision.io/buildrun, cloudivision.io/project
   - use runner image from CLOU_DIVISION_RUNNER_IMAGE or cloudivision/runner:dev
   - pass env vars BUILD_RUN_NAME, BUILD_RUN_NAMESPACE, PROJECT_NAME, REPOSITORY_URL, REVISION, BRANCH, PIPELINE_TEMPLATE_NAME, IMAGE_REPOSITORY, IMAGE_TAG, GITOPS_ENABLED
   - resource requests/limits come from PipelineTemplate.spec.resources
   - activeDeadlineSeconds from timeoutSeconds
   - backoffLimit default 0 or 1
   - ttlSecondsAfterFinished default 3600
   - container securityContext: runAsNonRoot, allowPrivilegeEscalation false, privileged false, drop all capabilities where possible
7. If Job exists:
   - active > 0 => Running
   - succeeded > 0 => Succeeded
   - failed and backoffLimit exhausted => Failed
8. Update BuildRun status: phase, startedAt, completedAt, jobRef, conditions, failure reason/message.
9. Emit Kubernetes Events for JobCreated, BuildStarted, BuildSucceeded and BuildFailed.

Rules:
- Reconciler must be idempotent.
- It must not create multiple Jobs for the same BuildRun.
- Use CreateOrUpdate or explicit existence checks.
- Do not use kubectl.
- Do not use a database.
- Add controller tests using fake client or envtest.

Acceptance criteria:
- go test ./... passes.
- BuildRun with valid references creates exactly one Job.
- Repeated reconcile does not duplicate Jobs.
- Job success updates BuildRun status to Succeeded.
- Job failure updates BuildRun status to Failed.
- Job spec does not use privileged, hostPath or docker.sock.
- RBAC markers grant only required permissions.
```

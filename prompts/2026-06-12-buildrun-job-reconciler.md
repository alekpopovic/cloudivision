# BuildRun Job reconciler

User prompt:

```text
Continue working on the cloudivision project.

Task:
Implement the BuildRun reconciler that creates and tracks a Kubernetes Job for every BuildRun with executor=job.

Reconcile behavior:
1. Load the BuildRun.
2. If the BuildRun is being deleted, run cleanup/finalizer logic if needed.
3. If the BuildRun does not have finalizer cloudivision.io/buildrun-finalizer, add it.
4. Validate references:
   - Project exists
   - Repository exists
   - PipelineTemplate exists
5. If BuildRun is already in a terminal phase:
   - Succeeded
   - Failed
   - Cancelled
   then do not create a new Job.
6. If no Job exists:
   - create a Kubernetes Job in the Project namespace or BuildRun namespace
   - Job must have ownerReference pointing to the BuildRun
   - Job labels:
     - app.kubernetes.io/name=cloudivision
     - app.kubernetes.io/component=runner
     - cloudivision.io/buildrun=<buildrun-name>
     - cloudivision.io/project=<project-name>
   - Job uses runner image from config:
     - CLOU_DIVISION_RUNNER_IMAGE
     - default cloudivision/runner:dev
   - Env vars for runner:
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
   - resource requests/limits come from PipelineTemplate.spec.resources
   - activeDeadlineSeconds comes from timeoutSeconds
   - backoffLimit defaults to 0 or 1
   - ttlSecondsAfterFinished defaults to 3600
   - securityContext:
     - runAsNonRoot true
     - allowPrivilegeEscalation false
     - privileged false
     - drop all capabilities where possible
7. If Job exists:
   - read Job status
   - if active > 0, set BuildRun status to Running
   - if succeeded > 0, set BuildRun status to Succeeded
   - if failed and backoffLimit is exhausted, set BuildRun status to Failed
8. Update BuildRun status:
   - phase
   - startedAt
   - completedAt
   - jobRef
   - conditions
   - failure reason/message
9. Emit Kubernetes Events for:
   - JobCreated
   - BuildStarted
   - BuildSucceeded
   - BuildFailed

Additional rules:
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
- RBAC markers grant only required permissions for BuildRuns, Jobs, Events and relevant CRDs.
```

# Controller reliability contract

cloudivision reconcilers calculate desired names deterministically and treat reconciliation as repeatable. This document records the reliability decisions implemented for Project, BuildRun and Release.

## Project

- Namespace, ServiceAccount, Role, RoleBinding and NetworkPolicy use `CreateOrUpdate`; existing unrelated labels are preserved.
- Project status is written only when phase, observed generation or Ready condition changes.
- A five-minute requeue repairs namespaced resources deleted externally. The interval avoids a hot loop and is needed because cross-namespace resources cannot carry a namespaced Project owner reference.
- Project uses no finalizer. It does not delete a team namespace or its contents when the Project is deleted.

## BuildRun

- Job/PipelineRun names derive from the BuildRun name. `EnsureRun` reads before create and treats AlreadyExists as success, so repeated reconcile creates one child.
- Jobs and Releases have a same-namespace controller owner reference and consistent `cloudivision.io/buildrun`/project labels. The controller watches both child kinds and recreates a deleted non-terminal Job or required Release.
- Terminal BuildRuns never create a new execution Job. A successful terminal run may still repair its single declarative Release.
- BuildRun no longer adds a finalizer because children use garbage-collection owner references and there is no external cleanup obligation. The controller retains conflict-safe removal of the legacy `cloudivision.io/buildrun-finalizer` so upgrades cannot strand deletions.
- Transition events are emitted once; a repeated Running observation does not spam `BuildStarted`.

## Release

- Release creation is deterministic per BuildRun/Environment.
- A successful external GitOps update is checkpointed in the Release annotation `cloudivision.io/gitops-commit` before status is written. If the status subresource conflicts or is unavailable, the next reconcile restores status from the checkpoint instead of creating another Git commit.
- GitOps transport failures and unavailable/non-healthy Argo CD state remain retryable in `Deploying`, with an explicit 30-second requeue. Rejected releases, policy violations and invalid references are terminal failures.
- `Deployed` and `RolledBack` releases are terminal and do not repeat external side effects.
- Release uses no finalizer: cloudivision does not delete Git commits or deployed workloads during Release deletion.

## Status conflicts and conditions

All controller/runner status writes use the status subresource and retry Kubernetes conflicts. On retry, desired conditions win by condition type while unrelated conditions from concurrent actors are preserved. `observedGeneration` is set from the object generation represented by each controller transition; spec is never written through a status update.

Unit tests cover repeated reconcile, existing/deleted children, terminal behavior, a real injected status conflict, Git commit success followed by status failure, missing Argo CD state and an existing Project namespace. Live API-server behavior is additionally covered by the conformance/upgrade harnesses when a cluster is available.

# Prompt 26 — Controller Reliability and Conflict Handling

Phase: Reliability

```text
Continue working on the cloudivision project.

Task:
Harden all Kubernetes controllers for reliability, idempotency and status update conflicts.

Controllers to review:
- Project controller
- BuildRun controller
- Release controller
- any additional controllers already implemented

For each controller, verify and improve:
1. Idempotency:
   - Reconcile can repeat without duplicate resources.
   - Existing resources are updated only when required.
   - Desired state calculation is separated from side effects where practical.
2. Status subresource usage:
   - Spec/status updates are separate.
   - Status patches do not overwrite spec.
   - Conditions use observedGeneration correctly.
   - Terminal phases are respected.
3. Conflict handling:
   - Use retry-on-conflict or patch patterns for status updates.
   - Handle resource version conflicts gracefully.
   - Avoid losing conditions during concurrent updates.
4. Finalizers:
   - Use only where cleanup is required.
   - Deletion path removes finalizer after cleanup.
   - Deletion path does not get stuck on non-critical cleanup.
5. OwnerReferences:
   - Jobs have correct ownerReference.
   - Labels and owner refs are consistent.
   - Cross-namespace ownerReference limitations are respected.
6. Requeue behavior:
   - Explicit durations for external state.
   - No hot loops on missing optional dependencies.
   - No permanent failure on transient errors.
7. Events:
   - Useful transitions only, no event spam.

Tests:
- repeated reconcile
- status update conflict
- child resource already exists
- child resource deleted externally
- terminal BuildRun does not create new Job
- Release GitOps commit succeeds but status update fails
- Argo CD Application missing
- Project namespace already exists

Acceptance criteria:
- go test ./... passes.
- Controller tests cover idempotency.
- Status updates are conflict-safe.
- Finalizers are justified and documented.
- Reconcile loops do not create duplicate Jobs or Releases.
- Events are meaningful but not spammy.
```

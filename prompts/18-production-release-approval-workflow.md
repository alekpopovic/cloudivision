# Prompt 18 — Production Release Approval Workflow

Phase: Release / GitOps

```text
Continue working on the cloudivision project.

Task:
Implement a basic approval workflow for Release in cloudivision.

Goal:
If Environment.spec.requiresApproval=true, Release must not update the GitOps repo until approved.

API endpoints:
- POST /api/v1/releases/{namespace}/{name}/approve
- POST /api/v1/releases/{namespace}/{name}/reject

Approve/reject request:
{
  "actor": "string",
  "comment": "string"
}

Behavior:
1. Release controller: if approval.required=true and approvedBy is empty, set status.phase=AwaitingApproval and do not perform GitOps update.
2. Approve endpoint patches Release.spec.approval.approvedBy and approvedAt, adds annotation/status audit marker and records audit event.
3. Reject endpoint sets status.phase=Failed or rejected field if model is extended, does not perform GitOps update and records audit event.
4. Angular UI Release detail shows Approve/Reject buttons for AwaitingApproval. If auth is incomplete, actor can come from request or dev-mode current user, clearly labelled.
5. Domain validation: deployed release cannot be approved/rejected again; rejected release cannot deploy; release without required approval should not show approval actions.

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Release AwaitingApproval does not modify GitOps repo.
- Approve allows next reconcile to perform GitOps update.
- Reject blocks release.
- Audit event is created for approve/reject.
- Angular UI displays basic approval flow.
```

# Pull-request-based GitOps promotion

Direct commits remain the default. Set `spec.promotionMode: pull-request` when a
Release must pass through repository review and branch protection before the
GitOps provider may sync it.

```yaml
spec:
  promotionMode: pull-request
  pullRequest:
    targetBranch: production
    titleTemplate: "Promote {{release.name}}"
    bodyTemplate: "Deploy {{image}} after review."
    reviewers: [platform-team]
    labels: [deployment, production]
```

The supported template variables are `{{release.name}}` and `{{image}}`. If
`targetBranch` is omitted, the BuildRun GitOps branch is used, falling back to
`main`. The controller creates or resets the deterministic branch
`cloudivision/<release-name>` from the target, commits the image update, and calls
`CreateOrUpdatePullRequest`. This deterministic branch and provider operation make
reconciliation idempotent: retrying does not open another PR/MR.

While review is open, the Release remains `GitOpsChangeCommitted` and exposes:

- `status.pullRequest.provider`, `url`, and `reference`
- `status.pullRequest.headBranch` and `targetBranch`
- `status.pullRequest.mergeStatus`

A merged PR moves to `WaitingForSync`; provider sync/health handling is then the
same as direct commit. A PR closed without merge moves to `FailedApproval`.
`direct-commit` bypasses all PR calls and preserves the existing flow.

## Provider setup

GitHub and GitLab provider skeletons are selected from the repository host. They
require an authenticated API client implementing the narrow pull-request API
interface. Until one is configured, the Release fails with a clear
`pull request provider is not configured` message. Other repository hosts return
an explicit unsupported-provider error. Tokens must come from scoped Kubernetes
Secrets, must not be written to status or logs, and should have access only to the
GitOps repository.

The web Release view links to the external PR/MR and shows promotion and merge
state. Review, approval, and merge remain in the source provider; cloudivision does
not duplicate a review UI.

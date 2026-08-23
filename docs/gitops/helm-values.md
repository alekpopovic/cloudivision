# Helm values GitOps releases

`helm-values` is the primary direct-commit GitOps strategy. After a successful
BuildRun, the Release controller clones the configured repository, checks out
the target branch, updates one values file, creates a release-specific commit,
pushes it, and records the resulting SHA in `Release.status.gitCommit`.

## Complete BuildRun configuration

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: BuildRun
metadata:
  name: checkout-main-20260823
  namespace: ci
spec:
  projectRef: payments
  repositoryRef: checkout
  pipelineTemplateRef: go-service
  revision: main
  triggeredBy:
    type: manual
    actor: release-operator
  image:
    repository: ghcr.io/acme/checkout
    tag: main-a1b2c3d
  executor: job
  gitOps:
    enabled: true
    repoURL: https://github.com/acme/platform-gitops.git
    branch: production
    path: apps/checkout
    strategy: helm-values
    environmentRef: production
    valuesFile: values-production.yaml
    imageRepositoryField: workloads.checkout.image.repository
    imageTagField: workloads.checkout.image.tag
    imageDigestField: workloads.checkout.image.digest
```

The four field settings default to `values.yaml`, `image.repository`,
`image.tag`, and `image.digest`. `gitOps.path` is a repository-relative directory;
the older form where `path` directly names a YAML file remains supported when
`valuesFile` is its default.

For the example above, this input:

```yaml
# Production checkout workload
workloads:
  checkout:
    image:
      repository: ghcr.io/acme/checkout # managed by cloudivision
      tag: previous
      digest: sha256:old
```

is updated at the configured nested keys. Comments and mapping order are retained
where supported by `yaml.v3`. A release digest updates `digest`; an absent desired
digest removes a stale digest. The desired tag is retained alongside a digest so
charts that use both fields continue to render correctly.

## Idempotency and failures

Before staging files, the provider compares the configured scalar values. If the
desired repository, tag, and digest are already present, it returns the current
HEAD without creating a commit or pushing. The controller also checkpoints the
commit in both Release status and an annotation, so a status-update retry does
not repeat the external mutation.

Failures are terminal and explicit:

| Stage | Release phase | Failure reason |
| --- | --- | --- |
| clone or checkout | `FailedGitClone` | `GitCloneFailed` |
| read or YAML parse | `FailedGitCommit` | `GitOpsParseFailed` |
| unsafe path or incompatible field shape | `FailedGitCommit` | `GitOpsUpdateFailed` |
| stage, commit, or read HEAD | `FailedGitCommit` | `GitCommitFailed` |
| push | `FailedGitPush` | `GitPushFailed` |

Paths are constrained to the cloned repository. A configured field path may
create missing mapping nodes, but it fails instead of replacing an existing
non-map or non-scalar value. Repository credentials must be supplied through the
Git runtime credential mechanism and must never be embedded in CR fields or
logged.

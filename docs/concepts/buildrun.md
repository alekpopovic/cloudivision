# BuildRun

A BuildRun is one immutable CI request for a repository revision and template. The controller creates exactly one Job (or optional Tekton PipelineRun), and stores phase, timestamps, artifact, evidence, logs summary, policy decision, and failure in status.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: BuildRun
metadata: {name: storefront-main-001, namespace: ci}
spec:
  projectRef: storefront
  repositoryRef: storefront
  pipelineTemplateRef: storefront-ci
  revision: main
  commitSHA: 0123456789abcdef
  triggeredBy: {type: manual, actor: developer}
  image: {repository: ghcr.io/acme/storefront, tag: main}
  executor: job
  gitOps: {enabled: false}
```

Terminal phases are `Succeeded`, `Failed`, and `Cancelled`. Retrying creates a new BuildRun; do not mutate a completed run. Kubernetes status is authoritative even when PostgreSQL audit storage is enabled.

The API and CLI expose lifecycle actions:

- `POST .../{name}/cancel` / `cloudivision build cancel NAME` stops the runner
  Job and marks Pending, Queued, or Running runs as Cancelled. Repeated cancel is
  safe.
- `POST .../{name}/retry` / `cloudivision build retry NAME` creates a new run
  only from Failed or Cancelled runs and records `cloudivision.io/retry-of`.
- `POST .../{name}/rerun` / `cloudivision build rerun NAME` creates a new run
  from any terminal run and records `cloudivision.io/rerun-of`.

The original resource remains terminal and unchanged, preserving its status,
logs, audit records, and relationship history.

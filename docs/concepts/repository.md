# Repository

A Repository binds source URL/provider, default branch, credential Secret reference, webhook configuration, Project, and PipelineTemplate. Secret values never belong in the CR.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Repository
metadata: {name: storefront, namespace: ci}
spec:
  projectRef: storefront
  provider: github
  url: https://github.com/acme/storefront.git
  defaultBranch: main
  credentialSecretRef: {name: storefront-git, key: token}
  pipelineTemplateRef: storefront-ci
  webhook:
    enabled: true
    secretRef: {name: storefront-webhook, key: secret}
    events: [push]
    branchFilters:
      include: [main, "release/*"]
      exclude: ["release/private-*"]
    tagFilters:
      include: ["v*"]
      exclude: ["v0.*"]
    pullRequest:
      enabled: true
      events: [opened, reopened, synchronize]
      buildForks: false
      requireTrustedActor: false
```

Supported source provider identifiers are `github`, `gitlab`, `gitea`, and `generic`. Health/capability metadata does not prove access to each private repository; verify credentials and egress separately.

Include and exclude entries accept exact names and shell-style glob patterns; exclude
always wins. If `branchFilters.include` is empty, only `defaultBranch` is built for
backward compatibility. Tags are disabled until `tagFilters.include` contains at
least one pattern. Pull requests are disabled unless `pullRequest.enabled` is true,
and only their configured actions and matching base branches are built. Fork builds
remain blocked unless `buildForks` is explicitly true. `requireTrustedActor` is a
fail-closed integration hook: until a provider identity is mapped to the auth model,
the event is ignored as untrusted.

Ignored events return HTTP 200 with `result: ignored` and a human-readable reason.
They also produce an auditable `WebhookAccepted` record whose metadata contains a
stable reason such as `branch_ignored`, `tag_disabled`, or
`fork_pull_request_blocked`.

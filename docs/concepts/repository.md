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
```

Supported source provider identifiers are `github`, `gitlab`, `gitea`, and `generic`. Health/capability metadata does not prove access to each private repository; verify credentials and egress separately.

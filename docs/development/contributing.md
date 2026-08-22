# Contributing

Start from an issue-sized change and preserve the platform boundaries documented
in [architecture](architecture.md). Inspect adjacent tests before editing.

```sh
git switch -c feature/short-description
go test ./...
go vet ./...
npm --prefix web run build
git status --short
```

Keep commits focused and use a concise imperative subject. Include tests for
meaningful behavior, update examples and references when an API or Helm value
changes, and state any cluster-only checks that were not run.

Security-sensitive changes must retain least privilege, restricted pod security,
secret redaction and webhook signature verification. New integrations belong
behind small interfaces; follow [adding a provider](adding-provider.md).

A pull request should explain the problem, design choice, verification performed,
operational impact and rollback or compatibility considerations.

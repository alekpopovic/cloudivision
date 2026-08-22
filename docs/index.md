# cloudivision documentation

cloudivision is a Kubernetes-native CI/CD platform: BuildRuns execute as Kubernetes Jobs, artifact evidence is stored in CR status, and Releases deploy by changing a GitOps repository rather than applying application manifests from CI.

## Start here

- [kind quickstart](getting-started/quickstart-kind.md)
- [first build](getting-started/first-build.md)
- [first webhook](getting-started/first-webhook.md)
- [first GitOps release](getting-started/first-release.md)
- [CLI reference](reference/cli.md)

## Learn the model

[Project](concepts/project.md) · [Repository](concepts/repository.md) · [PipelineTemplate](concepts/pipeline-template.md) · [BuildRun](concepts/buildrun.md) · [Environment](concepts/environment.md) · [Release](concepts/release.md) · [Provider](concepts/provider.md) · [Policy](concepts/policy.md)

## Operate and review

- Operators: [Helm install](operations/install-helm.md), [upgrade](operations/upgrade.md), [backup/restore](operations/backup-restore.md), [observability](operations/observability.md), [troubleshooting](operations/troubleshooting.md), and [hardening](operations/security-hardening.md).
- Security reviewers: [threat model](security/threat-model.md), [runner security](security/runner-security.md), [supply chain](security/supply-chain.md), [auth/RBAC](security/auth-rbac.md), and [webhooks](security/webhook-security.md).
- Contributors: [architecture](development/architecture.md), [controllers](development/controllers.md), [runner](development/runner.md), [API](development/api.md), [frontend](development/frontend.md), [testing](development/testing.md), and [contributing](development/contributing.md).
- Reference: [HTTP API](reference/api.md), [CRDs](reference/crds.md), [Helm values](reference/helm-values.md), and [configuration](reference/configuration.md).

Older detailed notes under `docs/guides`, `docs/api`, `docs/controllers`, `docs/install`, and ADRs remain linked as deeper implementation/history references. The pages above are the maintained user-facing entry points.

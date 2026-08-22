# Changelog

All notable user-facing changes are recorded here. Versions follow Semantic
Versioning; the Kubernetes API remains `v1alpha1` and has the compatibility
guarantees documented in `docs/api/compatibility-policy.md`.

## [Unreleased]

### Features

### Fixes

### Security

### Breaking changes

### Upgrade notes

### Known issues

## [0.1.0] - 2026-08-22

### Features

- Kubernetes-native Project, Repository, PipelineTemplate, BuildRun, Environment,
  and Release APIs with Job execution and an optional Tekton adapter.
- Go API, runner, controller manager, developer CLI, Angular/Tailwind UI, Helm
  chart, provider health, audit support, policy evaluation, and GitOps promotion.
- Conformance, upgrade, security, observability, and limited scale harnesses.

### Fixes

- Idempotent Job and Release reconciliation with explicit terminal diagnostics.
- API pagination and filters for bounded BuildRun history views.
- Correct production approval payloads and structured policy errors.

### Security

- Restricted non-root defaults, least-privilege runner RBAC, webhook signature
  verification and replay protection, secret redaction, workload limits, and
  opt-in supply-chain adapters.
- Public release artifacts include checksums; images include SBOM/provenance and
  keyless signatures in the GitHub release workflow.

### Breaking changes

- This is the initial public application release. The served CRD version is
  experimental `cicd.cloudivision.io/v1alpha1`; minor releases may require
  manifest migration according to the compatibility policy.

### Upgrade notes

- Fresh installation is recommended for the first public release.
- Helm does not upgrade CRDs from a chart's `crds/` directory. Future upgrades
  must back up custom resources and apply the reviewed CRD bundle before upgrading
  workloads.
- `Chart.yaml` `version` and `appVersion` are both `0.1.0`.

### Known issues

- Clean-cluster conformance, live upgrade, and sustained scale evidence must be
  reviewed in the final v0.1 release gate before publishing.
- SBOM/provenance files generated inside runner workspaces are not yet persisted
  automatically to an external artifact store.
- Tekton, Argo CD, Flux, signing, scanning, and PostgreSQL paths require explicit
  external configuration; several adapters are intentionally initial or optional.
- GitHub currently reports dependency alerts that must be triaged before a public
  production recommendation.

[Unreleased]: https://github.com/alekpopovic/cloudivision/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/alekpopovic/cloudivision/releases/tag/v0.1.0

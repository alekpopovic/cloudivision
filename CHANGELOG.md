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
- Replaced the quadratic PipelineTemplate step-name CEL rule with a structural
  list-map key so CRDs install within Kubernetes admission cost budgets.
- Corrected the chart Pod security context field and made the rootless nginx web
  image use a numeric UID and write its PID to `/tmp`; web health probes now keep
  traffic away until nginx is ready.
- Made pre-artifact BuildRun status omit `image`, preventing the CRD schema from
  rejecting controller and runner status updates before an image exists.
- Pointed the credential-free quickstart, conformance, upgrade, and scale fixtures
  at a verified public Node.js repository.
- Made the live upgrade harness use the portable deployment availability wait
  supported by current kubectl releases.
- Added per-project API RoleBindings to an otherwise unbound permission template
  for documented CR and pod-log access while retaining namespaced get-only
  (never list/watch) Secret permission.

### Security

- Restricted non-root defaults, least-privilege runner RBAC, webhook signature
  verification and replay protection, secret redaction, workload limits, and
  opt-in supply-chain adapters.
- Public release artifacts include checksums; images include SBOM/provenance and
  keyless signatures in the GitHub release workflow.
- CI and release validation fail on high or critical production web dependency
  advisories; the v0.1 production dependency tree currently audits clean.

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
- `PipelineTemplate.spec.steps` is a Kubernetes list-map keyed by `name`; manifests
  with duplicate step names must be corrected before installation or upgrade.

### Known issues

- Clean-cluster conformance, live upgrade, and sustained scale evidence must be
  reviewed in the final v0.1 release gate before publishing.
- SBOM/provenance files generated inside runner workspaces are not yet persisted
  automatically to an external artifact store.
- Tekton, Argo CD, Flux, signing, scanning, and PostgreSQL paths require explicit
  external configuration; several adapters are intentionally initial or optional.
- Six known npm audit findings remain in development-only Angular build tooling
  (three high, three moderate); resolving them requires a breaking Angular major
  upgrade and is tracked for v0.2. Production dependencies audit clean.

[Unreleased]: https://github.com/alekpopovic/cloudivision/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/alekpopovic/cloudivision/releases/tag/v0.1.0

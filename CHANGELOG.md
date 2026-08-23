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

## [0.2.1] - 2026-08-23

### Fixes

- Create the configured Go temporary directory before SDK generation, scale
  validation, and local release packaging so clean GitHub runners do not fail
  immediately with `GOTMPDIR` errors.
- Publish reviewed, version-specific release notes and a source SPDX SBOM from
  the release workflow, and use those notes as the GitHub Release body.

### Upgrade notes

- v0.2.1 is a packaging and CLI patch over v0.2.0. CRD schemas are unchanged;
  use the v0.2.1 chart, CRD bundle, CLI, and component images together.

### Known issues

- The v0.2.0 tag remains as immutable evidence of the failed first publication
  attempt; its workflow stopped before images, artifacts, or a GitHub Release
  were published. v0.2.1 is the installable release.

## [0.2.0] - 2026-08-23

### Features

- Added hardened BuildKit and registry flows, supply-chain evidence, dependency
  caches, artifact/log backends, quotas, cancellation, retry and concurrency.
- Expanded webhook filters, GitOps strategies, release promotion/rollback,
  notifications, organizations/teams, audit exports, compliance reports, and
  optional cluster targets and remote runners.
- Added richer Angular onboarding, debugging, release, provider, organization,
  report, and plugin views plus generated Go and TypeScript API clients.
- Added experimental `v1beta1` conversion and admission foundations, an
  in-process plugin registry, a versioned pipeline catalog, eight runnable
  examples, and expanded conformance/upgrade/scale coverage.
- Added the cloudivision visual identity and a searchable Jekyll documentation
  site with architecture, operational, security, API, and roadmap guidance.

### Fixes

- Wired `cloudivision release rollback` to the v0.2 API with explicit target,
  actor, and reason fields instead of returning the stale v0.1 error.
- Made scalar list CRD fields atomic so Kubernetes 1.36 does not reject their
  schemas for quadratic `uniqueItems` validation cost.
- Made conformance fail immediately on unexpected terminal BuildRun phases and
  use a checked-in real-world source fixture.
- Excluded generated example dependencies and output from product image build
  contexts.

### Security

- Added optional admission validation, organization-aware authorization,
  least-privilege multi-cluster and remote-runner designs, signed webhook replay
  protection, audit/report exports, and a CRD complexity security gate.
- Runner defaults remain non-root, bounded and unprivileged with no Docker socket
  or hostPath dependency; production npm dependencies audit clean.

### Breaking changes

- The stored Kubernetes API remains experimental `v1alpha1`. New fields are
  additive, but Helm does not upgrade CRDs automatically; apply the reviewed
  v0.2 CRD bundle before upgrading workloads.
- Primitive build platform, dependency-cache path, restore-key, and raw-YAML file
  lists now use atomic server-side-apply semantics.

### Upgrade notes

- Back up custom resources, apply `charts/cloudivision/crds/` or `config/crd/bases/`,
  then upgrade controller, API, runner, and web images together.
- The live same-code lifecycle gate preserved custom resources and reconciled a
  new BuildRun. A true published v0.1.0-to-v0.2.0 rehearsal is still recommended.

### Known issues

- The Job executor still runs step commands inside the runner image and does not
  honor per-step images; Tekton supports step images. This remains planned API
  and executor work.
- Rootless BuildKit with a private authenticated registry and an external GitOps
  commit were not exercised in the v0.2 local final gate.
- Six development-only Angular toolchain advisories remain (three moderate and
  three high); production dependencies have no reported vulnerabilities.
- Optional admission webhooks, remote runners, cluster targets and `v1beta1`
  conversions are experimental and disabled by default.

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

[Unreleased]: https://github.com/alekpopovic/cloudivision/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/alekpopovic/cloudivision/releases/tag/v0.2.1
[0.2.0]: https://github.com/alekpopovic/cloudivision/releases/tag/v0.2.0
[0.1.0]: https://github.com/alekpopovic/cloudivision/releases/tag/v0.1.0

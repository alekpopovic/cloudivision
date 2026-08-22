# Product backlog after v0.1

This backlog implements the [v0.2 roadmap](v0.2.md). Priority is based on closing
the verified v0.1 user-path gaps, not on feature count. Items inside each category
are ordered; the team should finish a vertical slice before starting another.

## v0.2 must-have

### M1. Authenticated rootless image push and digest capture

- **Problem:** no final-gate run built and pushed an image to a private registry;
  downstream deployment can still depend on mutable tags.
- **Proposed solution:** complete the rootless BuildKit adapter, Secret-based
  registry auth, bounded push retry, manifest verification, and digest status.
- **Acceptance criteria:** disposable-registry integration test passes; status and
  Release input use `repository@sha256:...`; transient retry is idempotent; no
  credential reaches logs or status.
- **Risks:** BuildKit/storage compatibility and registry-specific error handling.
- **Dependencies:** builder interface, runner Job security, image status schema,
  registry fixture.

### M2. Durable GitHub event policy

- **Problem:** signature/idempotency tests exist, but restart-safe replay control
  and precise branch/PR/push behavior are not production-proven.
- **Proposed solution:** durable delivery records with TTL, explicit normalized
  event types, branch filters, timestamp replay checks, and rejection metrics.
- **Acceptance criteria:** restart/redelivery integration tests create one
  BuildRun; excluded/stale/invalid events create none; push and PR metadata differ
  predictably.
- **Risks:** false replay rejection and unbounded delivery-record retention.
- **Dependencies:** PostgreSQL idempotency storage, Repository trigger config,
  webhook provider fixtures.

### M3. Digest-only direct GitOps commit

- **Problem:** v0.1 skipped the real writable-repository assertion.
- **Proposed solution:** harden direct commit and `helm-values` mutation with path
  allow-listing, optimistic conflict handling, and persisted commit checkpoints.
- **Acceptance criteria:** a real test repository receives exactly one scoped
  digest update; retry reuses the commit; conflicts have actionable status; no
  target-cluster apply occurs from CI.
- **Risks:** unintended file mutation and duplicate/conflicting commits.
- **Dependencies:** M1 digest, Git credential Secret, Git adapter, Release status.

### M4. Argo CD health-driven Release completion

- **Problem:** Release success is not proven against actual deployment health.
- **Proposed solution:** add a read-only Argo CD application reader and map sync,
  health, degraded, missing, and timeout states to Release conditions.
- **Acceptance criteria:** integration tests cover Synced/Healthy, Degraded,
  provider unavailable, and deadline exceeded; reconciliation remains idempotent.
- **Risks:** eventual consistency and version differences in Argo CD status.
- **Dependencies:** M3 commit, Argo CD test instance and read-only RBAC, deployment
  timeout configuration.

### M5. Stage-oriented build and release diagnostics

- **Problem:** users must correlate separate views to locate registry or GitOps
  failures.
- **Proposed solution:** expose structured stages/correlation IDs from the API and
  render them in BuildRun and Release detail, with bounded logs and safe retries.
- **Acceptance criteria:** browser tests cover clone, build, push, policy, commit,
  sync, and timeout failures; API error code/message remains visible; pagination
  and log bounds are enforced.
- **Risks:** UI/backend state drift and excessive polling.
- **Dependencies:** M1–M4 status contracts, log API, provider health API.

### M6. Repository onboarding readiness checks

- **Problem:** configuration errors are often discovered only after the first
  failed webhook or push.
- **Proposed solution:** extend the wizard with webhook-secret, repository-access,
  registry-auth, branch-filter, and provider-health validation.
- **Acceptance criteria:** onboarding can run non-destructive checks, identifies
  the failing dependency, never returns secret material, and has empty/error/
  forbidden browser coverage.
- **Risks:** validation endpoints could become credential probes or diverge from
  runtime behavior.
- **Dependencies:** M1/M2 adapters, authz, redaction, provider health.

### M7. Least-privilege and network conformance

- **Problem:** v0.1 fixed cross-namespace API access during the gate, but provider
  egress and production OIDC were not live-tested.
- **Proposed solution:** continuously assert the unbound API permission template,
  audit used verbs, test cross-project OIDC denial, and publish tested NetworkPolicy
  examples for DNS/Git/registry/telemetry egress.
- **Acceptance criteria:** security checks fail on global template binding or broad
  Secret verbs; OIDC and NetworkPolicy conformance pass; documented Roles match
  observed calls.
- **Risks:** provider-specific endpoints and accidental denial of required traffic.
- **Dependencies:** CNI-capable test cluster, OIDC issuer, audit logs, security
  scripts.

### M8. v0.1-to-v0.2 release qualification

- **Problem:** v0.1 could only run a same-version upgrade and GitHub reports 64
  dependency alerts whose applicability is not represented by local npm runtime
  audit alone.
- **Proposed solution:** run the live upgrade from the published v0.1 artifacts;
  inventory Go/npm/container/workflow alerts; fix exploitable findings; document
  scoped exceptions; execute external integration conformance and a measured
  100-BuildRun run.
- **Acceptance criteria:** pre/post-upgrade resources survive; required external
  scenarios have no skips; zero unexplained critical/high production findings;
  latency/resource results and exceptions are attached to the release review.
- **Risks:** upstream advisories or external service flakiness can delay release.
- **Dependencies:** published v0.1 artifacts, dependency scanners, M1–M7 test
  environments, metrics-server.

## v0.2 nice-to-have

### N1. Flux health reader

- **Problem:** Flux users cannot see deployment convergence through the same
  Release state model.
- **Proposed solution:** add a read-only Flux adapter matching the Argo CD health
  interface after that contract stabilizes.
- **Acceptance criteria:** Ready, failed, missing, and timeout fixtures map to
  provider-neutral conditions with no write permission.
- **Risks:** premature abstraction and Flux object/version variation.
- **Dependencies:** M4 provider interface and a Flux integration fixture.

### N2. Pull-request promotion polish

- **Problem:** PR mode exists, but reviewer-facing context and merge-state recovery
  are less complete than direct commit mode.
- **Proposed solution:** enrich deterministic PRs with digest, policy, BuildRun,
  and environment context and reconcile merged/closed state.
- **Acceptance criteria:** retries reuse one PR; UI links it; merged, closed, and
  provider-error states are clear and audited.
- **Risks:** provider permission and branch-protection differences.
- **Dependencies:** M1 digest, GitHub/GitLab adapters, Release state machine.

### N3. API latency sampling in scale gate

- **Problem:** v0.1's live limited scale run lacked API/log latency and resource
  samples.
- **Proposed solution:** provision metrics-server and API access in the disposable
  scale environment and emit machine-readable thresholds.
- **Acceptance criteria:** CI stores p50/p95 list/log latency, controller/API pod
  resources, and terminal counts for 10 and 100 BuildRuns.
- **Risks:** noisy shared-runner measurements.
- **Dependencies:** scale harness, metrics-server, API auth fixture.

### N4. Angular build-tool major upgrade

- **Problem:** six development-only npm findings remain on Angular 20 tooling.
- **Proposed solution:** evaluate the supported Angular major migration separately,
  update tests/config, and remove deprecated animation usage.
- **Acceptance criteria:** full npm audit has no high finding or an approved
  time-bounded exception; build and 22+ browser tests pass.
- **Risks:** migration churn could distract from the core v0.2 vertical slice.
- **Dependencies:** Angular migration tooling and stable UX test coverage.

## v0.3 candidates

### C1. Stable CRD beta migration

- **Problem:** `v1alpha1` limits compatibility expectations for serious operators.
- **Proposed solution:** use collected upgrade evidence to introduce `v1beta1`,
  conversion, storage migration, and deprecation policy.
- **Acceptance criteria:** round-trip conversion and skew tests pass; rollback and
  migration runbooks are rehearsed; no stored object data is lost.
- **Risks:** conversion webhook availability and irreversible schema choices.
- **Dependencies:** v0.2 upgrade evidence and compatibility policy.

### C2. Persistent artifact metadata and retention

- **Problem:** runner SBOM/provenance files are not automatically persisted.
- **Proposed solution:** add an artifact-store interface, immutable metadata index,
  retention policy, and signed download references.
- **Acceptance criteria:** artifacts survive Job cleanup, access is authorized, and
  retention/deletion is auditable.
- **Risks:** storage cost, sensitive data, and provider lock-in.
- **Dependencies:** digest identity, authz, audit backend, supply-chain adapters.

### C3. Policy bundles and admission integration

- **Problem:** code-based policy is useful but hard to delegate across teams.
- **Proposed solution:** versioned policy bundles with dry-run evaluation and an
  optional Kubernetes admission integration.
- **Acceptance criteria:** bundle changes are validated/audited; dry-run explains
  impact; controller and API produce the same decision.
- **Risks:** policy language complexity and availability coupling.
- **Dependencies:** policy decision schema, authz, audit/event storage.

### C4. Sustained scale and log architecture

- **Problem:** short tests do not establish endurance or large-log behavior.
- **Proposed solution:** 1,000-BuildRun soak tests, bounded log streaming/storage,
  backpressure, and explicit SLO dashboards.
- **Acceptance criteria:** documented 24-hour run meets error/latency/resource
  budgets and 100 MB logs cannot exhaust API/controller memory.
- **Risks:** infrastructure cost and environment-dependent results.
- **Dependencies:** N3 measurements, external log store, observability stack.

## Future enterprise features

### E1. Organization/team tenancy and SSO lifecycle

- **Problem:** file-based group mapping does not cover enterprise identity
  lifecycle or delegated administration.
- **Proposed solution:** persistent organizations/teams, SCIM provisioning,
  delegated roles, and audited access reviews.
- **Acceptance criteria:** tenant isolation and deprovisioning tests pass; every
  permission change is attributable and reversible.
- **Risks:** complex authorization model and compliance obligations.
- **Dependencies:** stable API/auth model, PostgreSQL HA, audit retention.

### E2. Multi-cluster execution and deployment fleet

- **Problem:** large organizations need isolated regional/build clusters.
- **Proposed solution:** explicit cluster registration, scoped credentials,
  scheduling policy, health inventory, and failure-domain-aware execution.
- **Acceptance criteria:** compromise of one cluster credential cannot access
  another; scheduling and failover are observable and policy-controlled.
- **Risks:** large security blast radius and distributed-state complexity.
- **Dependencies:** stable single-cluster SLOs, tenancy, secret management.

### E3. Compliance evidence and long-term audit

- **Problem:** regulated teams need exportable evidence beyond operational logs.
- **Proposed solution:** tamper-evident audit retention, approval/evidence bundles,
  policy attestations, and controlled exports.
- **Acceptance criteria:** a release evidence bundle links identity, source,
  digest, policy, approvals, deployment, and signatures with retention controls.
- **Risks:** privacy, storage cost, legal retention, and cryptographic key custody.
- **Dependencies:** persistent audit/artifacts, SSO identity, supply-chain signing.

### E4. Availability, support, and disaster-recovery tiers

- **Problem:** production customers require measurable service objectives and
  rehearsed recovery.
- **Proposed solution:** HA topology, backup automation, regional recovery tests,
  SLO reporting, and supported upgrade windows.
- **Acceptance criteria:** published RPO/RTO and availability targets are met in
  repeated failure drills with no CR ownership ambiguity.
- **Risks:** operational cost and false confidence from synthetic drills.
- **Dependencies:** PostgreSQL HA, multi-cluster design, observability, support
  process.

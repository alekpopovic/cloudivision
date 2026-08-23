# cloudivision Prompt Tracker

Project: `cloudivision`
Created at: `2026-08-21`
Updated at: `2026-08-23T10:32:09Z`
Last executed prompt: `65`
Next prompt: `66`

## How to update

Run:

```bash
python3 scripts/mark_prompt_done.py 21
```

Replace `21` with the last prompt that was successfully executed.

## Status legend

- `done` — Prompt has been executed successfully.
- `in_progress` — Prompt is currently being executed.
- `blocked` — Prompt cannot continue until a blocker is resolved.
- `pending` — Prompt has not been executed yet.
- `skipped` — Prompt was intentionally skipped.

## Prompt status

| ID | Status | Phase | Title | File | Notes |
|---:|---|---|---|---|---|
| 00 | done | Foundation | Master Codex Prompt: AGENTS.md | `prompts/00-master-codex-prompt-agents-md.md` | Marked done because user stated prompts 0-20 were completed. |
| 01 | done | Foundation | Repository Scaffold | `prompts/01-repository-scaffold.md` | Marked done because user stated prompts 0-20 were completed. |
| 02 | done | Kubernetes API | Kubebuilder / Operator Foundation | `prompts/02-kubebuilder-operator-foundation.md` | Marked done because user stated prompts 0-20 were completed. |
| 03 | done | Kubernetes API | Detailed CRD Models | `prompts/03-detailed-crd-models.md` | Marked done because user stated prompts 0-20 were completed. |
| 04 | done | Build Execution | BuildRun Controller: Kubernetes Job Executor | `prompts/04-buildrun-controller-kubernetes-job-executor.md` | Marked done because user stated prompts 0-20 were completed. |
| 05 | done | Build Execution | Runner Binary: Clone, Steps, Build Image, Push | `prompts/05-runner-binary-clone-steps-build-image-push.md` | Marked done because user stated prompts 0-20 were completed. |
| 06 | done | API | API Server: REST API Over CRDs | `prompts/06-api-server-rest-api-over-crds.md` | Marked done because user stated prompts 0-20 were completed. |
| 07 | done | API | Git Webhook Service: GitHub, GitLab, Gitea, Generic | `prompts/07-git-webhook-service-github-gitlab-gitea-generic.md` | Marked done because user stated prompts 0-20 were completed. |
| 08 | done | Release / GitOps | GitOps Integration and Release Controller | `prompts/08-gitops-integration-and-release-controller.md` | Marked done because user stated prompts 0-20 were completed. |
| 09 | done | Executor | Executor Interface and Tekton Adapter | `prompts/09-executor-interface-and-tekton-adapter.md` | Marked done because user stated prompts 0-20 were completed. |
| 10 | done | Security | Multi-Tenancy, RBAC and Pod Security Hardening | `prompts/10-multi-tenancy-rbac-and-pod-security-hardening.md` | Marked done because user stated prompts 0-20 were completed. |
| 11 | done | Audit | PostgreSQL Audit and Cached UI View | `prompts/11-postgresql-audit-and-cached-ui-view.md` | Marked done because user stated prompts 0-20 were completed. |
| 12 | done | Frontend | Web UI: Angular + Tailwind Dashboard | `prompts/12-web-ui-angular-tailwind-dashboard.md` | Marked done because user stated prompts 0-20 were completed. |
| 13 | done | Installation | Helm Chart and Cluster Installation | `prompts/13-helm-chart-and-cluster-installation.md` | Marked done because user stated prompts 0-20 were completed. |
| 14 | done | Installation | Local Dev: kind Quickstart and Demo App | `prompts/14-local-dev-kind-quickstart-and-demo-app.md` | Marked done because user stated prompts 0-20 were completed. |
| 15 | done | Observability | Observability: Metrics, Events, Tracing-Ready Logs | `prompts/15-observability-metrics-events-tracing-ready-logs.md` | Marked done because user stated prompts 0-20 were completed. |
| 16 | done | Testing | Test Strategy: Unit, Envtest, kind E2E | `prompts/16-test-strategy-unit-envtest-kind-e2e.md` | Marked done because user stated prompts 0-20 were completed. |
| 17 | done | Security | Supply Chain Security: SBOM, Signing, Provenance Hooks | `prompts/17-supply-chain-security-sbom-signing-provenance-hooks.md` | Marked done because user stated prompts 0-20 were completed. |
| 18 | done | Release / GitOps | Production Release Approval Workflow | `prompts/18-production-release-approval-workflow.md` | Marked done because user stated prompts 0-20 were completed. |
| 19 | done | Auth | Auth and API/UI RBAC | `prompts/19-auth-and-api-ui-rbac.md` | Marked done because user stated prompts 0-20 were completed. |
| 20 | done | Review | Final Review Prompt: Full Project Audit | `prompts/20-final-review-prompt-full-project-audit.md` | Marked done because user stated prompts 0-20 were completed. |
| 21 | done | v0.1 Hardening | v0.1 Product Readiness Audit | `prompts/21-v0-1-product-readiness-audit.md` | Readiness audit completed; safe documentation and Make help fixes added. Clean-cluster runtime remains unverified and is documented as a blocker. |
| 22 | done | v0.1 Hardening | Conformance Test Suite | `prompts/22-conformance-test-suite.md` | Conformance suite, fixtures, diagnostics and documentation added. Static validation passed; live suite not run because the configured Kubernetes API endpoint refused connections. |
| 23 | done | Dogfood | Dogfood cloudivision With cloudivision | `prompts/23-dogfood-cloudivision-with-cloudivision.md` | Dogfood manifests, runnable Go/web BuildRuns, non-root toolchain runner and documentation added. Docker image build and local gates passed; live Kubernetes run unavailable. |
| 24 | done | API Evolution | CRD Compatibility and Versioning Plan | `prompts/24-crd-compatibility-and-versioning-plan.md` | Compatibility policy, v1alpha1 reference, v1beta1 plan, non-installing conversion topology and upgrade test skeleton added. |
| 25 | done | API Evolution | CRD Validation Hardening With Defaults and CEL | `prompts/25-crd-validation-hardening-with-defaults-and-cel.md` | CRD required/default/enum/CEL validation hardened; generated bases and Helm bundle synchronized; invalid admission fixtures and schema drift tests added. |
| 26 | done | Reliability | Controller Reliability and Conflict Handling | `prompts/26-controller-reliability-and-conflict-handling.md` | Controllers hardened for idempotency, conflict-safe conditions, child recreation, retryable external state and GitOps commit checkpointing; finalizer decisions documented and tests expanded. |
| 27 | done | Security | Runner Threat Model and Security Conformance | `prompts/27-runner-threat-model-and-security-conformance.md` | Added runner threat model and rendered Helm security conformance checks for privileged containers, docker.sock, hostPath, and broad RBAC. |
| 28 | done | Release / GitOps | Release State Machine Hardening | `prompts/28-release-state-machine-hardening.md` | Hardened Release phases, idempotent Git checkpointing, explicit Git/provider failures, deployment timeout, approval metadata, events, CRDs, tests, and lifecycle docs. |
| 29 | done | Release / GitOps | PR-Based GitOps Promotion | `prompts/29-pr-based-gitops-promotion.md` | Added optional PR-based GitOps promotion, deterministic branches, GitHub/GitLab provider skeletons, idempotent PR reconciliation, CRD/status metadata, UI link, docs, and tests. |
| 30 | done | Supply Chain | Real Supply Chain Adapters | `prompts/30-real-supply-chain-adapters.md` | Added opt-in Syft, Grype, Cosign and JSON provenance adapters, adapter CRD config, severity policy enforcement, secure key projection, BuildRun conditions/status, supply-chain UI tab, docs, and tests. |
| 31 | done | Frontend | Angular UX Upgrade for Real CI/CD Debugging | `prompts/31-angular-ux-upgrade-for-real-ci-cd-debugging.md` | Upgraded Angular debugging UX with BuildRun timelines/failures/reruns, advanced logs, repository onboarding, pipeline editor, Release detail, first-run workflow, routes, and tests. |
| 32 | done | Providers | Provider Adapter Registry | `prompts/32-provider-adapter-registry.md` | Added thread-safe provider registry and category packages, initial provider catalog, provider list/health APIs, Providers UI, OpenAPI/docs, and tests. |
| 33 | done | Policy | Policy Engine Layer | `prompts/33-policy-engine-layer.md` | Added centralized code-based policy evaluator, structured API denials, controller PolicyDenied status, CRD policy decisions, Angular violation UI, tests, and docs. |
| 34 | done | Operations | Upgrade, Backup, Restore and Uninstall | `prompts/34-upgrade-backup-restore-and-uninstall.md` | Added lifecycle operations runbooks plus offline/live Helm upgrade test, fixtures, CRD retention assertions, and Make target. |
| 35 | done | Observability | Observability Dashboards and Alerts | `prompts/35-observability-dashboards-and-alerts.md` | Added Grafana dashboard, Prometheus alerts, bounded operational metrics, controller metrics Service, runbooks, and OpenTelemetry tracing plan. |
| 36 | done | Scale | Performance and Scale Test Harness | `prompts/36-performance-and-scale-test-harness.md` | Added safe scale harness, BuildRun pagination/filtering, bounded Angular list rendering, controller indexes, and configurable concurrency. |
| 37 | done | CLI | cloudivision CLI | `prompts/37-cloudivision-cli.md` | Added cloudivision CLI with config/auth precedence, resource/build/release commands, watch/logs, doctor, tests, build target, and docs. |
| 38 | done | Documentation | Documentation Restructure for Users and Operators | `prompts/38-documentation-restructure-for-users-and-operators.md` | Restructured user, operator, security, contributor, and reference documentation; fixed outdated release approval payload. Local Markdown link and Helm render validation passed. |
| 39 | done | Release | Public Release Process and v0.1.0 | `prompts/39-public-release-process-and-v0-1-0.md` | Added semver release metadata, changelog/checklist, local artifact builder, GHCR image/signing workflow, checksums, Helm/CRD packaging, and public registry defaults. Local 0.1.0 artifact and checksum smoke test passed; Syft/Cosign were unavailable locally and explicitly skipped. |
| 40 | done | Release | v0.1 Final Gate: Hardening Sprint Review | `prompts/40-v0-1-final-gate-hardening-sprint-review.md` | Final gate passed on a clean kind cluster after safe CRD, Helm, web, status-schema, fixture, API RBAC, upgrade-harness, and Angular dependency fixes. Go/vet/web/Helm/security passed; conformance 5/5 with one external GitOps skip, live upgrade passed, limited scale 10/10; recommendation is ship v0.1.0 alpha with known issues. |
| 41 | done | Roadmap | v0.2 Roadmap Planning | `prompts/41-v0-2-roadmap-planning.md` | Created a focused v0.2 roadmap and ordered backlog tied to the v0.1 final-gate gaps: authenticated immutable image builds, durable GitHub webhooks, real GitOps/Argo convergence, stage-oriented Angular UX, security closure, cross-version upgrade, dependency triage, and measured scale. Deferred v0.3 and enterprise scope is explicit. |
| 42 | done | Roadmap | v0.2 Implementation Plan | `prompts/42-v0-2-implementation-plan.md` | Created the executable v0.2 implementation plan with seven epics, work items, CRD/API impacts, tests, acceptance criteria, risks, dependencies, milestone ordering, and explicit exclusions. |
| 43 | done | Build Execution | BuildKit Builder Hardening | `prompts/43-buildkit-builder-hardening.md` | Hardened rootless BuildKit input/path validation, deterministic command options, cache settings, bounded failures, metadata digest capture, runner image-stage conditions/status, CRDs, docs, and tests. Go test/vet and Helm security checks passed. |
| 44 | done | Providers | Registry Provider and Credentials | `prompts/44-registry-provider-and-credentials.md` | Added registry provider interface and generic/GHCR/GitLab/Harbor implementations, cloud skeletons, image/digest resolution, credential parsing/redaction, scoped Secret validation/projection, runner Docker config handling, CRDs/API models, docs, and tests. Go, Angular, Helm, and security gates passed. |
| 45 | done | Supply Chain | Image Tagging, Digest and Immutability | `prompts/45-image-tagging-digest-and-immutability.md` | Implemented deterministic safe image tag templates, digest-first GitOps updates, production latest/digest policy, separate UI tag/digest fields, tests and documentation. |
| 46 | done | API | GitHub Webhook Production Readiness | `prompts/46-github-webhook-production-readiness.md` | Hardened GitHub webhooks with HMAC verification, 1 MiB limit, push/PR/ping parsing, replay and delivery-ID idempotency, stable responses, bounded audit events, PostgreSQL migrations, tests and setup/security docs. |
| 47 | done | API | Branch, Tag and Pull Request Event Filters | `prompts/47-branch-tag-and-pull-request-event-filters.md` | Added auditable branch/tag/PR webhook filters, safe fork/trusted-actor defaults, Angular configuration/summary, CRD schemas, tests and docs. go test ./... and Angular build passed. |
| 48 | done | Build Execution | BuildRun Retry, Cancel and Rerun | `prompts/48-buildrun-retry-cancel-and-rerun.md` | Added idempotent BuildRun cancel plus immutable retry/rerun API actions, audit events, API RBAC, CLI commands, safe Angular actions/relations, tests and docs. Go tests, Angular build and Helm render passed. |
| 49 | done | Storage | Log Storage Abstraction | `prompts/49-log-storage-abstraction.md` | Added pluggable logstore interface with Kubernetes/default, memory/local, object/Loki skeletons, pre-persistence redaction, runner/API integration, BuildRun log status, Helm configuration, UI backend display, tests and operations docs. Go tests, Angular build and Helm renders passed. |
| 50 | done | Storage | Artifact Storage | `prompts/50-artifact-storage.md` | Added artifact storage abstraction with disabled/noop, memory/local, object/OCI skeleton backends; safe runner collection, SHA-256 metadata in BuildRun status, API list/download, Angular Artifacts tab, Helm local PVC config, demo pipeline, tests and docs. Go tests/vet, Angular tests/build, Helm renders and YAML validation passed. |
| 51 | done | Scale | Project Quotas and Runner Concurrency | `prompts/51-project-quotas-and-runner-concurrency.md` | Added enforceable project concurrency, queue, resource, duration, artifact, and log quotas with status/metrics/UI/docs coverage. |
| 52 | done | API | API Pagination and Server-Side Filtering | `prompts/52-api-pagination-and-server-side-filtering.md` | Added safe token pagination and server-side filtering/sorting for BuildRuns, Releases and audit events; Angular and CLI page navigation; API/OpenAPI docs and tests. |
| 53 | done | Build Execution | Runner Workspace and Dependency Cache | `prompts/53-runner-workspace-and-dependency-cache.md` | Added opt-in scoped dependency cache with PVC snapshots, BuildKit registry mode, object-store skeleton, TTL/size/path hardening, runner restore/save, purge API/UI, Helm configuration, tests and security docs. |
| 54 | done | Release / GitOps | GitOps Helm Values Production Flow | `prompts/54-gitops-helm-values-production-flow.md` | Hardened Helm-values GitOps with configurable values/field paths, comment-preserving YAML node updates, true no-op behavior, scoped failure reasons, idempotency/push tests and complete docs. |
| 55 | done | Release / GitOps | GitOps Kustomize and Raw YAML Support | `prompts/55-gitops-kustomize-and-raw-yaml-support.md` | Added targeted idempotent Kustomize and raw-YAML GitOps configuration/mutators for custom files, image names, workload kinds/names and containers, with traversal safety, multi-workload tests, CRDs and docs. |
| 56 | done | Release / GitOps | Argo CD and Flux Status Integration | `prompts/56-argo-cd-and-flux-status-integration.md` | Implemented unstructured Argo CD and Flux Kustomization/HelmRelease status readers, provider/resource absence conditions, deployment revision/time fields, UI, RBAC, tests, and docs. |
| 57 | done | Release / GitOps | Release Promotion and Rollback | `prompts/57-release-promotion-and-rollback.md` | Added immutable release promotion and rollback endpoints, target policy/approval handling, audit events, lineage fields/UI, controller rollback phase, tests, OpenAPI, and docs. |
| 58 | done | Providers | Notifications | `prompts/58-notifications.md` | Implemented Project-scoped notification configuration, generic webhook delivery with filtering/retries/redaction, Slack/Teams/email skeletons, controller/API event dispatch, failure conditions, provider health UI, tests, and docs. |
| 59 | done | Security | Secret Provider Integrations | `prompts/59-secret-provider-integrations.md` | Implemented selected-key Kubernetes Secret provider with namespace enforcement/redaction, integrated registry/webhook/notification resolution, minimal runner projections, External Secrets CRD health detection, Vault skeleton, tests, RBAC, UI presence, and docs. |
| 60 | done | Templates | Pipeline Template Catalog | `prompts/60-pipeline-template-catalog.md` | Added versioned 11-template pipeline catalog, deploy manifests and per-template docs, catalog list/install APIs, Angular catalog install/import UI, and backend/frontend tests. |
| 61 | done | Frontend | Angular Pipeline Editor Upgrade | `prompts/61-angular-pipeline-editor-upgrade.md` | Upgraded Angular PipelineTemplate editor with full step/build/security controls, reorder/remove, catalog import, YAML preview, duplicate and unsafe validation, backend create/update validation, API update endpoint, and tests. |
| 62 | done | Frontend | Angular First-Run and Onboarding Polish | `prompts/62-angular-first-run-and-onboarding-polish.md` | Polished eight-stage first-run wizard with empty detection, progress, catalog selection/install, optional webhook, first BuildRun trigger/log link, copyable API command, next-step docs, dashboard redirect, validation/error tests. |
| 63 | done | CLI | CLI Packaging and Distribution | `prompts/63-cli-packaging-and-distribution.md` | Added version/commit/date metadata, bash/zsh/fish/PowerShell completions, Make targets, verified install script, GoReleaser config, six-platform CLI release artifacts, installation docs, and config/API/error/JSON tests. Full Go tests/vet and cross-compilation passed. Added version/commit/date metadata, bash/zsh/fish/PowerShell completions, Make targets, verified install script, GoReleaser config, six-platform CLI release artifacts, installation docs, and config/API/error/JSON tests. Full Go tests/vet and CLI builds passed. |
| 64 | done | Installation | Helm Chart Production Hardening | `prompts/64-helm-chart-production-hardening.md` | Added production Helm scheduling and annotation controls, read-only runtime filesystems, startup probes, optional PDB/HPA/ServiceMonitor/TLS/NetworkPolicy customization, leader-election guidance, safe extra env/volume escape hatches, render matrix tests, and complete operator/value documentation. Helm lint/template matrix, security checks, rendered YAML parsing, and full Go tests passed. |
| 65 | done | API Evolution | CRD v1beta1 API Implementation | `prompts/65-crd-v1beta1-api-implementation.md` | Added experimental v1beta1 types, loss-preserving alpha/beta conversions, round-trip tests, and API documentation; v1alpha1 remains the served storage version. |
| 66 | pending | API Evolution | Admission Webhooks and Defaulting | `prompts/66-admission-webhooks-and-defaulting.md` |  |
| 67 | pending | Architecture | Multi-Cluster Architecture | `prompts/67-multi-cluster-architecture.md` |  |
| 68 | pending | Architecture | Remote Runner / Agent Mode | `prompts/68-remote-runner-agent-mode.md` |  |
| 69 | pending | Multi-Tenancy | Organizations, Teams and Memberships | `prompts/69-organizations-teams-and-memberships.md` |  |
| 70 | pending | Audit | Audit Export and Compliance Reports | `prompts/70-audit-export-and-compliance-reports.md` |  |
| 71 | pending | API | OpenAPI, SDK and API Client Generation | `prompts/71-openapi-sdk-and-api-client-generation.md` |  |
| 72 | pending | Extensibility | Plugin System v1 | `prompts/72-plugin-system-v1.md` |  |
| 73 | pending | Examples | More Real-World Examples | `prompts/73-more-real-world-examples.md` |  |
| 74 | pending | Release | v0.2 Final Gate | `prompts/74-v0-2-final-gate.md` |  |
| 75 | pending | Roadmap | v0.3 Roadmap Planning | `prompts/75-v0-3-roadmap-planning.md` |  |

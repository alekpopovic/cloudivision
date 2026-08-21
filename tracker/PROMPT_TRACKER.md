# cloudivision Prompt Tracker

Project: `cloudivision`
Created at: `2026-08-21`
Updated at: `2026-08-21`
Last executed prompt: `20`
Next prompt: `21`

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
| 21 | pending | v0.1 Hardening | v0.1 Product Readiness Audit | `prompts/21-v0-1-product-readiness-audit.md` |  |
| 22 | pending | v0.1 Hardening | Conformance Test Suite | `prompts/22-conformance-test-suite.md` |  |
| 23 | pending | Dogfood | Dogfood cloudivision With cloudivision | `prompts/23-dogfood-cloudivision-with-cloudivision.md` |  |
| 24 | pending | API Evolution | CRD Compatibility and Versioning Plan | `prompts/24-crd-compatibility-and-versioning-plan.md` |  |
| 25 | pending | API Evolution | CRD Validation Hardening With Defaults and CEL | `prompts/25-crd-validation-hardening-with-defaults-and-cel.md` |  |
| 26 | pending | Reliability | Controller Reliability and Conflict Handling | `prompts/26-controller-reliability-and-conflict-handling.md` |  |
| 27 | pending | Security | Runner Threat Model and Security Conformance | `prompts/27-runner-threat-model-and-security-conformance.md` |  |
| 28 | pending | Release / GitOps | Release State Machine Hardening | `prompts/28-release-state-machine-hardening.md` |  |
| 29 | pending | Release / GitOps | PR-Based GitOps Promotion | `prompts/29-pr-based-gitops-promotion.md` |  |
| 30 | pending | Supply Chain | Real Supply Chain Adapters | `prompts/30-real-supply-chain-adapters.md` |  |
| 31 | pending | Frontend | Angular UX Upgrade for Real CI/CD Debugging | `prompts/31-angular-ux-upgrade-for-real-ci-cd-debugging.md` |  |
| 32 | pending | Providers | Provider Adapter Registry | `prompts/32-provider-adapter-registry.md` |  |
| 33 | pending | Policy | Policy Engine Layer | `prompts/33-policy-engine-layer.md` |  |
| 34 | pending | Operations | Upgrade, Backup, Restore and Uninstall | `prompts/34-upgrade-backup-restore-and-uninstall.md` |  |
| 35 | pending | Observability | Observability Dashboards and Alerts | `prompts/35-observability-dashboards-and-alerts.md` |  |
| 36 | pending | Scale | Performance and Scale Test Harness | `prompts/36-performance-and-scale-test-harness.md` |  |
| 37 | pending | CLI | cloudivision CLI | `prompts/37-cloudivision-cli.md` |  |
| 38 | pending | Documentation | Documentation Restructure for Users and Operators | `prompts/38-documentation-restructure-for-users-and-operators.md` |  |
| 39 | pending | Release | Public Release Process and v0.1.0 | `prompts/39-public-release-process-and-v0-1-0.md` |  |
| 40 | pending | Release | v0.1 Final Gate: Hardening Sprint Review | `prompts/40-v0-1-final-gate-hardening-sprint-review.md` |  |
| 41 | pending | Roadmap | v0.2 Roadmap Planning | `prompts/41-v0-2-roadmap-planning.md` |  |

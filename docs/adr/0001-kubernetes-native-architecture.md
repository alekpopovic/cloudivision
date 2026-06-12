# 0001 Kubernetes-native architecture

## Status

Accepted.

## Context

cloudivision needs to coordinate CI/CD workflows inside Kubernetes while keeping runtime behavior observable, recoverable, and aligned with Kubernetes control-loop patterns.

## Decision

cloudivision uses Kubernetes CRDs as the source of truth for runtime state. Pipeline, build, release, and environment progress must be represented in Kubernetes resource status instead of being stored only in a database.

The first build executor uses Kubernetes Jobs. Tekton support will be added later through an executor adapter so the domain model is not hardcoded to Tekton PipelineRuns.

CD goes through GitOps. The CI runner creates artifacts and updates GitOps repositories; it does not directly apply application manifests to target namespaces in the MVP. Argo CD or Flux reconciles desired state into the cluster.

The frontend uses Angular, TypeScript, and Tailwind CSS.

## Consequences

- Controllers must be idempotent and safe to re-run.
- PostgreSQL can support audit logs, cached UI views, webhook idempotency, and future user/team metadata, but it is not the primary runtime state store.
- Build workloads must avoid privileged defaults, Docker socket mounting, and Docker-in-Docker assumptions.
- Executor implementations must remain behind small interfaces.


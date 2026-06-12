# cloudivision

cloudivision is an early-stage, open-source Kubernetes-native CI/CD platform. It is designed to run inside a Kubernetes cluster and model CI/CD workflows with Kubernetes APIs, controllers, Jobs, and GitOps deployment integrations.

## Architecture

The platform is planned as a monorepo with these major parts:

- Kubernetes API types under `api/v1alpha1`
- a controller/operator under `cmd/controller` and `internal/controller`
- an API server under `cmd/api`
- a build runner under `cmd/runner`
- executor abstractions under `internal/executor`
- build, Git, GitOps, webhook, auth, audit, and Kubernetes helper packages under `internal`
- an Angular + Tailwind CSS web UI under `web`
- Helm and Kubernetes installation assets under `charts` and `config`

Runtime state for pipelines, builds, and releases belongs in Kubernetes custom resource status. PostgreSQL may be added later for audit logs, cached UI views, webhook idempotency, and user/team metadata, but it is not required for the first scaffold.

## MVP flow

The initial product flow is:

```text
webhook -> API -> BuildRun CR -> controller -> Kubernetes Job -> registry -> GitOps repo -> Argo CD/Flux
```

CI creates artifacts. CD happens through GitOps by updating deployment repositories and letting Argo CD or Flux reconcile the target environments. The CI runner must not directly apply application manifests to target namespaces in the MVP.

## Status

cloudivision is in early development. The current repository contains the initial scaffold and placeholder entrypoints for the API server, controller, runner, Helm chart, docs, and Angular web application.

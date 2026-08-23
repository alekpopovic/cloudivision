<p align="center">
  <img src="web/public/assets/brand/cloudivision-mark.png" alt="cloudivision modular C logo" width="150">
</p>

<h1 align="center">cloudivision</h1>

<p align="center">
  <strong>Kubernetes-native CI/CD, from verified source event to observable GitOps release.</strong>
</p>

<p align="center">
  Pipelines are custom resources. Builds run as isolated Kubernetes Jobs.<br>
  Deployments happen through Git—not from the CI runner.
</p>

<p align="center">
  <a href="https://github.com/alekpopovic/cloudivision/actions/workflows/ci.yaml"><img src="https://github.com/alekpopovic/cloudivision/actions/workflows/ci.yaml/badge.svg" alt="CI status"></a>
  <a href="https://github.com/alekpopovic/cloudivision/actions/workflows/docs-pages.yaml"><img src="https://github.com/alekpopovic/cloudivision/actions/workflows/docs-pages.yaml/badge.svg" alt="Documentation build status"></a>
  <img src="https://img.shields.io/badge/release-v0.2.0-4F7CFF" alt="Release v0.2.0">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26">
  <img src="https://img.shields.io/badge/Angular-20-DD0031?logo=angular&logoColor=white" alt="Angular 20">
  <img src="https://img.shields.io/badge/Kubernetes-native-326CE5?logo=kubernetes&logoColor=white" alt="Kubernetes native">
</p>

> [!IMPORTANT]
> v0.2.0 is an alpha release. The clean-cluster gate passes, but the default
> disabled authentication mode is for development only. Read the
> [final gate](docs/readiness/v0.2-final-gate.md) before production evaluation.

## How it works

<p align="center">
  <img src="docs/assets/brand/architecture.svg" alt="cloudivision architecture: Git event to BuildRun, isolated runner Job, OCI image, GitOps repository, and Argo CD or Flux deployment" width="100%">
</p>

The Kubernetes API remains the source of runtime truth. Controllers reconcile
desired state idempotently, the runner creates artifacts, and GitOps controllers
own target-cluster deployment.

```mermaid
stateDiagram-v2
    [*] --> Pending: webhook or manual trigger
    Pending --> Running: Job / PipelineRun created
    Running --> Succeeded: steps + image evidence complete
    Running --> Failed: bounded diagnostic captured
    Succeeded --> ReleasePending: GitOps enabled
    ReleasePending --> Deploying: commit or PR recorded
    Deploying --> Released: provider healthy
    Deploying --> Failed: degraded or timed out
```

## Why cloudivision

| | Capability | What it gives you |
| --- | --- | --- |
| <img src="web/public/assets/brand/icons/platform.svg" alt="" width="42"> | **Kubernetes-native state** | Projects, repositories, pipelines, builds, environments, and releases live as CRDs with status. |
| <img src="web/public/assets/brand/icons/pipeline.svg" alt="" width="42"> | **Pluggable execution** | Kubernetes Jobs are the MVP executor; Tekton remains behind the same interface. |
| <img src="web/public/assets/brand/icons/build.svg" alt="" width="42"> | **Artifact-first CI** | Rootless build paths, immutable image evidence, policy, SBOM, scanning, signing, and provenance hooks. |
| <img src="web/public/assets/brand/icons/deploy.svg" alt="" width="42"> | **GitOps delivery** | Direct commits or pull requests update deployment repositories; Argo CD and Flux can report health. |
| <img src="web/public/assets/brand/icons/security.svg" alt="" width="42"> | **Secure defaults** | Non-root workloads, no Docker socket, no default privilege, namespaced runner RBAC, signed webhooks, and redaction. |
| <img src="web/public/assets/brand/icons/observe.svg" alt="" width="42"> | **Operational visibility** | Angular debugging UI, CLI, structured conditions, logs, metrics, events, audit adapters, dashboards, and alerts. |

## Quickstart on kind

Prerequisites: Docker, kind, kubectl, Helm, Go 1.26+, and Node.js 24/npm.

```sh
git clone https://github.com/alekpopovic/cloudivision.git
cd cloudivision
./hack/kind-create.sh
./hack/kind-load-images.sh
./hack/install-dev.sh
```

Run the credential-free sample:

```sh
kubectl apply \
  -f deploy/examples/project.yaml \
  -f deploy/examples/repository.yaml \
  -f deploy/examples/pipeline-template-nodejs.yaml \
  -f deploy/examples/environment-dev.yaml \
  -f deploy/examples/buildrun-manual.yaml

kubectl -n cloudivision get buildrun demo-buildrun-manual -w
kubectl -n cloudivision logs \
  -l cloudivision.io/buildrun=demo-buildrun-manual \
  --tail=100
```

The sample clones Docker's public getting-started Node.js application and verifies
its source without requiring BuildKit or registry credentials. Continue with the
[full kind quickstart](docs/getting-started/quickstart-kind.md) or
[create your first build](docs/getting-started/first-build.md).

### Open the API and UI

```sh
# terminal 1
kubectl -n cloudivision port-forward \
  svc/cloudivision-cloudivision-api 8080:8080

# terminal 2
kubectl -n cloudivision port-forward \
  svc/cloudivision-cloudivision-web 4200:80
```

- API health: `http://localhost:8080/healthz`
- Angular UI: `http://localhost:4200`

Optional CLI check:

```sh
go build -o bin/cloudivision ./cmd/cloudivision
CLOU_DIVISION_API_URL=http://localhost:8080 \
  bin/cloudivision -n cloudivision doctor
```

## Platform map

| Component | Responsibility | Source |
| --- | --- | --- |
| Controller | Reconciles Projects, BuildRuns, Jobs, Releases, RBAC, and network policy | [`cmd/controller`](cmd/controller), [`internal/controller`](internal/controller) |
| API | REST facade, webhook intake, auth, logs, providers, and audit | [`cmd/api`](cmd/api), [`internal/api`](internal/api) |
| Runner | Clones source, executes steps, builds/pushes images, and records evidence | [`cmd/runner`](cmd/runner), [`internal/runner`](internal/runner) |
| CLI | Developer and operator workflows over the API | [`cmd/cloudivision`](cmd/cloudivision), [`internal/cli`](internal/cli) |
| Web | Angular + TypeScript + Tailwind operations UI | [`web`](web) |
| APIs | `cicd.cloudivision.io/v1alpha1` domain and generated CRDs | [`api/v1alpha1`](api/v1alpha1), [`config/crd`](config/crd) |
| Packaging | Helm chart, example resources, and local kind automation | [`charts/cloudivision`](charts/cloudivision), [`deploy/examples`](deploy/examples), [`hack`](hack) |

## Security model

cloudivision treats repository code as untrusted:

- runner workloads are non-root and resource-bounded;
- privileged containers, hostPath, Docker-in-Docker, and Docker socket mounts are
  not defaults;
- Project namespaces receive scoped ServiceAccounts, Roles, and RoleBindings;
- the API permission template is bound only inside reconciled Project namespaces;
- webhook signatures are checked before BuildRun creation;
- credentials and known secret values are redacted from logs and status; and
- CI runs rendered Helm security checks and a production dependency audit.

Start with the [threat model](docs/security/threat-model.md),
[runner security](docs/security/runner-security.md), and
[operator hardening guide](docs/operations/security-hardening.md).

## Documentation

Browse the modern, searchable [cloudivision documentation site](https://alekpopovic.github.io/cloudivision/) or use the source links below.

| Start | Build and release | Operate | Extend |
| --- | --- | --- | --- |
| [Documentation home](docs/index.md) | [First build](docs/getting-started/first-build.md) | [Helm install](docs/operations/install-helm.md) | [Architecture](docs/development/architecture.md) |
| [kind quickstart](docs/getting-started/quickstart-kind.md) | [First webhook](docs/getting-started/first-webhook.md) | [Upgrade](docs/operations/upgrade.md) | [Add a provider](docs/development/adding-provider.md) |
| [CLI reference](docs/reference/cli.md) | [First GitOps release](docs/getting-started/first-release.md) | [Troubleshooting](docs/operations/troubleshooting.md) | [Contributing](docs/development/contributing.md) |
| [CRD reference](docs/reference/crds.md) | [Policy model](docs/concepts/policy.md) | [Observability](docs/operations/observability.md) | [Brand system](docs/brand.md) |

## Release status

- Version: **v0.2.0 alpha**
- Readiness decision: **ship with known issues**
- Next increment: v0.3 roadmap
- Prioritized work: [product backlog](docs/roadmap/backlog.md)
- Changes: [CHANGELOG.md](CHANGELOG.md)

Useful local gates:

```sh
mise exec -- go test ./...
mise exec -- go vet ./...
npm --prefix web ci
npm --prefix web run build
helm template cloudivision charts/cloudivision --include-crds
make security-check
```

Contributions should preserve the core boundary: **CI creates artifacts; CD
deploys through GitOps.** See [contributing](docs/development/contributing.md) and
the [release process](docs/development/release-process.md).

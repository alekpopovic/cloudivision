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

## Local kind quickstart

This quickstart runs cloudivision in a local kind cluster. It installs the CRDs, controller, API server, and Angular web UI with the Helm chart.

### Prerequisites

- Docker
- kind
- kubectl
- Helm
- Go 1.26 or newer
- Node.js/npm for building the Angular web image

### Create the kind cluster

```sh
./hack/kind-create.sh
```

The script creates a `cloudivision-dev` kind cluster and starts a local registry at `localhost:5001` when Docker is available.

### Build and load images

```sh
./hack/kind-load-images.sh
```

This builds:

- `ghcr.io/cloudivision/controller:dev`
- `ghcr.io/cloudivision/api:dev`
- `ghcr.io/cloudivision/runner:dev`
- `ghcr.io/cloudivision/web:dev`

and loads them into the kind cluster.

### Install cloudivision

```sh
./hack/install-dev.sh
```

Check the installed pods:

```sh
kubectl -n cloudivision get pods
kubectl -n cloudivision get crds | grep cloudivision
```

### Apply sample CRs

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f deploy/examples/repository.yaml
kubectl apply -f deploy/examples/pipeline-template-nodejs.yaml
kubectl apply -f deploy/examples/environment-dev.yaml
kubectl apply -f deploy/examples/buildrun-manual.yaml
```

The default sample repository points at this public repository and verifies the checked-in `deploy/demo-app` files. For a real app test, push or fork `deploy/demo-app` to a Git repository, then update `deploy/examples/repository.yaml` to use that URL.

### Check the BuildRun

```sh
kubectl -n cloudivision get buildruns
kubectl -n cloudivision describe buildrun demo-buildrun-manual
kubectl -n cloudivision get jobs,pods
```

View runner logs:

```sh
kubectl -n cloudivision logs -l cloudivision.io/buildrun=demo-buildrun-manual --tail=100
```

### Open the API and Angular web UI

In one terminal:

```sh
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-api 8080:8080
```

In another terminal:

```sh
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-web 4200:80
```

Then open:

- API health: `http://localhost:8080/healthz`
- Web UI: `http://localhost:4200`

The Helm chart writes the web runtime config to `/assets/config.json`. For local port-forwarding, the default empty `apiBaseUrl` lets the UI call the same origin. If you serve the UI separately, set `web.config.apiBaseUrl` in Helm values.

## Troubleshooting

### CRD not installed

Run:

```sh
helm template cloudivision charts/cloudivision --include-crds
kubectl get crds | grep cicd.cloudivision.io
```

Reinstall with:

```sh
./hack/install-dev.sh
```

### Runner image cannot be pulled

Load images into kind again:

```sh
./hack/kind-load-images.sh
kubectl -n cloudivision describe pod -l cloudivision.io/buildrun=demo-buildrun-manual
```

For kind, `imagePullPolicy: IfNotPresent` is expected. The chart and examples use the `dev` tag by default.

### BuildKit not available

The quickstart pipeline has `build.enabled: false`, so it does not require BuildKit. If you enable image builds, the runner image or build environment must provide `buildctl` or `buildctl-daemonless.sh`; otherwise the runner fails with setup guidance.

### RBAC forbidden

Check the controller and runner ServiceAccounts:

```sh
kubectl -n cloudivision logs deploy/cloudivision-cloudivision-controller
kubectl -n cloudivision get role,rolebinding,serviceaccount
```

The Project controller creates namespaced runner RBAC for the project namespace. Re-apply `deploy/examples/project.yaml` if the namespace or ServiceAccount is missing.

### Pod Security rejection

The sample Project requests `restricted` Pod Security labels. Describe the rejected pod or namespace:

```sh
kubectl describe namespace cloudivision
kubectl -n cloudivision describe pod -l cloudivision.io/buildrun=demo-buildrun-manual
```

Keep runner workloads non-root and avoid privileged containers, hostPath mounts, and Docker socket mounts.

### Git credentials missing

The quickstart uses a public repository and does not require credentials. Private repositories need a Kubernetes Secret referenced from `Repository.spec.credentialSecretRef`.

### Angular UI cannot reach API

For local port-forwarding, either open the UI through the web service and use same-origin API routing, or set:

```sh
helm upgrade --install cloudivision charts/cloudivision \
  --namespace cloudivision \
  --set web.config.apiBaseUrl=http://localhost:8080
```

Then restart the web port-forward.

### CORS blocked in local development

Allow the Angular origin in Helm values:

```sh
helm upgrade --install cloudivision charts/cloudivision \
  --namespace cloudivision \
  --set api.cors.allowedOrigins='{http://localhost:4200,http://localhost:4201}'
```

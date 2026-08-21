# Dogfood cloudivision with cloudivision

The dogfood setup runs cloudivision's own formatting, tests, builds, Helm rendering and image-build definitions as cloudivision resources. It uses only local Kubernetes capacity and public container/source registries by default; no paid service is required.

## Current executor constraint

The Kubernetes Job executor currently clones the source once and executes every `PipelineTemplate.spec.steps` command inside the runner container. It records `step.image` but does not create a separate container for each step. The checked-in dogfood runner therefore includes Go, Node/npm, Chromium and Helm so the Go, web and release pipelines can run today. The images pipeline correctly declares rootless BuildKit images and commands, but it remains a known limitation until the Job executor honors per-step images or supplies rootless BuildKit in an isolated sidecar. It never mounts `docker.sock` and does not request privileged execution.

## Prepare the dogfood runner

Build and load the extended non-root runner into kind:

```sh
docker build -f build/dogfood-runner.Dockerfile \
  -t ghcr.io/cloudivision/dogfood-runner:dev .
kind load docker-image --name cloudivision-dev \
  ghcr.io/cloudivision/dogfood-runner:dev
helm upgrade cloudivision charts/cloudivision \
  --namespace cloudivision --reuse-values \
  --set runner.image.repository=dogfood-runner \
  --set runner.image.tag=dev
```

The chart combines `global.imageRegistry` with the runner repository, so the resulting Job image is `ghcr.io/cloudivision/dogfood-runner:dev`. The image runs as UID 65532 and does not need cluster-admin, privilege escalation, a hostPath or Docker socket.

## Configure and install resources

The checked-in Repository URL is `https://github.com/alekpopovic/cloudivision.git`. To use a fork, edit it before applying or patch it afterward:

```sh
kubectl apply -f deploy/dogfood/project.yaml
kubectl apply -f deploy/dogfood/repository.yaml
kubectl -n cloudivision-dogfood patch repository cloudivision \
  --type merge -p '{"spec":{"url":"https://github.com/YOUR_ORG/cloudivision.git"}}'
kubectl apply -f deploy/dogfood/pipeline-template-go.yaml
kubectl apply -f deploy/dogfood/pipeline-template-web.yaml
kubectl apply -f deploy/dogfood/pipeline-template-images.yaml
kubectl apply -f deploy/dogfood/pipeline-template-release.yaml
kubectl apply -f deploy/dogfood/environment-dev.yaml
```

Private forks need a Kubernetes Secret referenced by `Repository.spec.credentialSecretRef`. Do not commit registry or Git credentials. Create secrets directly in `cloudivision-dogfood` and grant only the dogfood runner ServiceAccount the existing namespaced read access.

## Trigger builds

Run the Go pipeline, which is the minimal fully local dogfood gate:

```sh
kubectl create -f deploy/dogfood/buildrun-go.yaml
```

Run the Angular pipeline:

```sh
kubectl create -f deploy/dogfood/buildrun-web.yaml
```

To trigger another template without adding a permanent manifest, start from the Go BuildRun and change its generated name/template/image:

```sh
sed 's/cloudivision-go-/cloudivision-release-/; s/pipelineTemplateRef: cloudivision-go/pipelineTemplateRef: cloudivision-release/' \
  deploy/dogfood/buildrun-go.yaml | kubectl create -f -
```

For `cloudivision-images`, first provide a runner execution path with `buildctl-daemonless.sh`, then set `PUSH_IMAGES=true` and the registry/tag values in the template if pushes are wanted. Registry credentials must be supplied through Kubernetes Secrets; the checked-in default does not push.

## Inspect status and logs

```sh
kubectl -n cloudivision-dogfood get projects,repositories,pipelinetemplates,buildruns,environments,releases
kubectl -n cloudivision-dogfood get jobs,pods
kubectl -n cloudivision-dogfood describe buildrun BUILD_RUN_NAME
kubectl -n cloudivision-dogfood logs \
  -l cloudivision.io/buildrun=BUILD_RUN_NAME --tail=200
```

With the API reachable at port 8080:

```sh
curl -fsS http://localhost:8080/api/v1/build-runs/cloudivision-dogfood/BUILD_RUN_NAME
curl -fsS http://localhost:8080/api/v1/build-runs/cloudivision-dogfood/BUILD_RUN_NAME/logs
```

The API needs intentionally scoped access to the dogfood namespace. Do not solve a `403` by granting cluster-admin; add a namespaced Role/RoleBinding for the API ServiceAccount according to the deployment's tenancy model.

## Optional GitOps release

Set `BuildRun.spec.gitOps.enabled: true`, `environmentRef: cloudivision-dev`, a writable GitOps repository URL, branch, path and strategy. A successful BuildRun creates a Release. Inspect it with:

```sh
kubectl -n cloudivision-dogfood get releases
kubectl -n cloudivision-dogfood describe release RELEASE_NAME
```

The generic provider performs a real Git clone, commit and push. The target repository must already contain the selected Helm values, Kustomize or raw YAML path and be reachable from the controller.

## What the pipelines cover

- `cloudivision-go`: gofmt drift, all Go tests, vet and all three binaries.
- `cloudivision-web`: locked npm install, production Angular build and configured browser tests.
- `cloudivision-images`: controller, API, runner and web Dockerfiles through rootless BuildKit definitions; optional push.
- `cloudivision-release`: Helm render plus rejection of privileged workloads, Docker sockets and hostPath volumes; optional GitOps promotion through the BuildRun.

All dogfood resources are namespaced. None creates a ClusterRole, ClusterRoleBinding or cluster-admin grant.

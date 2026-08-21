# Create your first BuildRun

This guide assumes cloudivision was installed with the local kind quickstart in the repository README and that the current kubectl context points to that cluster.

Apply the resources in dependency order:

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f deploy/examples/repository.yaml
kubectl apply -f deploy/examples/pipeline-template-nodejs.yaml
kubectl apply -f deploy/examples/environment-dev.yaml
kubectl apply -f deploy/examples/buildrun-manual.yaml
```

The Project controller prepares the project namespace and runner RBAC. The BuildRun controller then creates one namespaced Kubernetes Job for `demo-buildrun-manual`.

Watch the run and its workload:

```sh
kubectl -n cloudivision get buildrun demo-buildrun-manual -w
kubectl -n cloudivision get jobs,pods -l cloudivision.io/buildrun=demo-buildrun-manual
kubectl -n cloudivision logs -l cloudivision.io/buildrun=demo-buildrun-manual --tail=100
```

When the API is port-forwarded to port 8080, inspect the resource and logs without direct pod access:

```sh
curl -fsS http://localhost:8080/api/v1/build-runs/cloudivision/demo-buildrun-manual
curl -fsS http://localhost:8080/api/v1/build-runs/cloudivision/demo-buildrun-manual/logs
```

A successful run has `status.phase: Succeeded`, non-empty `startedAt` and `completedAt`, and a `Succeeded=True` condition. For failures, inspect `status.failure.reason`, `status.failure.message`, the Job events and runner logs. The checked-in sample only executes steps; image building is disabled so the first run does not depend on BuildKit or a registry credential.

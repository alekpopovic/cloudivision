# kind quickstart

Prerequisites: Docker, kind, kubectl, Helm, Go 1.26+, and Node.js 24/npm.

```sh
git clone https://github.com/alekpopovic/cloudivision.git
cd cloudivision
./hack/kind-create.sh
./hack/kind-load-images.sh
./hack/install-dev.sh
kubectl -n cloudivision rollout status deployment --all --timeout=5m
kubectl get crd | grep cicd.cloudivision.io
```

Apply and watch the working sample:

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f deploy/examples/repository.yaml
kubectl apply -f deploy/examples/pipeline-template-nodejs.yaml
kubectl apply -f deploy/examples/environment-dev.yaml
kubectl apply -f deploy/examples/buildrun-manual.yaml
kubectl -n cloudivision get buildrun demo-buildrun-manual -w
kubectl -n cloudivision logs -l cloudivision.io/buildrun=demo-buildrun-manual --tail=100
```

Expose the API and UI in separate terminals:

```sh
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-api 8080:8080
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-web 4200:80
curl -fsS http://localhost:8080/readyz
```

Open `http://localhost:4200`. The sample verifies a checked-out Node.js demo without building/pushing an image, so it does not require BuildKit or registry credentials.

If a pod does not start, run `kubectl -n cloudivision describe pod POD` and check image loading, Pod Security, quota, and runner ServiceAccount. Continue with [first build](first-build.md) or the [troubleshooting guide](../operations/troubleshooting.md).

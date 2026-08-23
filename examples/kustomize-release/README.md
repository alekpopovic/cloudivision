# Kustomize release example

A base Deployment and Service with a dev overlay, security settings, and illustrative Release resource.

```sh
cd examples/kustomize-release
sh test.sh
kubectl kustomize overlays/dev
```

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/kustomize-release/cloudivision.yaml
kubectl apply -f examples/kustomize-release/release.yaml
```

Replace the example Release digest and create its Environment before reconciliation. Rendering is local and requires no paid service.

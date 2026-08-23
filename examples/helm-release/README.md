# Helm release example

A secure Deployment chart plus CI and Release fixtures. The release is illustrative: replace the image digest and create an `example-production` Environment before applying its document.

```sh
cd examples/helm-release
sh test.sh
helm template example chart --set image.repository=example.invalid/api --set image.tag=local
```

Apply the build resources during evaluation. After replacing the illustrative digest and creating the referenced Environment, apply the Release separately:

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/helm-release/cloudivision.yaml
kubectl apply -f examples/helm-release/release.yaml
```

The chart is local and the default image reference is illustrative; no paid service is required.

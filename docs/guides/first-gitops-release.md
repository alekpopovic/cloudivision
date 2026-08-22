# Create your first GitOps release

cloudivision CI creates artifacts; deployment is performed by a GitOps controller. The runner does not apply application manifests directly to target namespaces.

First configure an Environment and a BuildRun whose `spec.gitOps.enabled` is true. The examples provide a starting point:

```sh
kubectl apply -f deploy/examples/environment-dev.yaml
kubectl apply -f deploy/examples/buildrun.yaml
```

The BuildRun GitOps block must name a strategy and repository location. A minimal example is:

```yaml
gitOps:
  enabled: true
  strategy: helm-values
  environmentRef: demo-dev
  repoURL: https://github.com/example/deployments.git
  branch: main
  path: apps/demo/values.yaml
```

After a successful build, inspect the generated Release:

```sh
kubectl -n cloudivision get buildruns,releases
kubectl -n cloudivision describe release RELEASE_NAME
```

Production Environments can require approval. Approve only after reviewing the artifact and supply-chain status through the API:

```sh
curl -fsS -X POST \
  http://localhost:8080/api/v1/releases/cloudivision/RELEASE_NAME/approve \
  -H 'Content-Type: application/json' \
  -d '{"actor":"local-operator","comment":"validated for development"}'
```

The current GitOps provider abstraction records Release progress and can read Argo CD state when configured. A real Git commit requires Git credentials and provider configuration appropriate to the target repository; confirm the commit and Argo CD/Flux sync independently before treating the release as deployed.

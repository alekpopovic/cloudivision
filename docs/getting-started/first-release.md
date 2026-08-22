# First GitOps release

Create an Environment and make a successful BuildRun GitOps-enabled. Replace the repository and path with a disposable writable GitOps repository:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Environment
metadata: {name: production, namespace: cloudivision}
spec:
  projectRef: demo-project
  displayName: Production
  namespace: demo-production
  type: production
  requiresApproval: true
  gitOps: {provider: argocd, applicationName: demo, namespace: argocd}
  policy: {requireImageDigest: true}
```

Helm values promotion uses `strategy: helm-values` with `path: apps/demo/values.yaml`; Kustomize uses `strategy: kustomize-image` with `path: apps/demo`. Configure the BuildRun:

```yaml
gitOps:
  enabled: true
  repoURL: https://github.com/example/gitops.git
  branch: main
  path: apps/demo/values.yaml
  strategy: helm-values
  environmentRef: production
```

Apply, observe, approve, and verify:

```sh
kubectl apply -f environment-production.yaml
kubectl apply -f buildrun-release.yaml
kubectl -n cloudivision get buildruns,releases -w
cloudivision --api-url http://localhost:8080 -n cloudivision release approve RELEASE_NAME --actor local-operator --comment "digest and evidence reviewed"
kubectl -n cloudivision get release RELEASE_NAME -o yaml
```

The controller commits desired state; Argo CD/Flux performs deployment. If the release remains `WaitingForSync`, inspect provider health and the GitOps application. Never work around a failure by applying target manifests from the runner.

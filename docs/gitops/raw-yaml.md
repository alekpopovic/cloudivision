# Raw YAML image releases

The `raw-yaml` strategy updates selected Kubernetes workload manifests without
running `kubectl apply`. It supports `Deployment`, `StatefulSet`, `DaemonSet`,
and `CronJob` pod templates, including multi-document YAML files.

```yaml
spec:
  gitOps:
    enabled: true
    repoURL: https://github.com/acme/platform-gitops.git
    branch: main
    path: clusters/production/checkout
    strategy: raw-yaml
    environmentRef: production
    rawYaml:
      files:
        - deployment.yaml
        - workers.yaml
      workloadKind: Deployment
      workloadName: checkout-api
      containerName: api
```

`files` are resolved relative to `gitOps.path`. Without `files`, the configured
path is treated as a file or recursively scanned for `.yaml` and `.yml` files.
The optional workload kind/name filters are applied before container matching.

Set `containerName` for multi-container workloads. Only that regular or init
container is changed; sidecars and other workloads remain untouched. When the
container name is omitted, cloudivision updates containers already using the
desired repository. It falls back to the only container in a single-container
workload, but refuses an ambiguous multi-container manifest. This conservative
behavior prevents an application release from rewriting an unrelated proxy or
metrics image.

The new value is `repository@digest` when a digest exists and otherwise
`repository:tag`. Invalid YAML reports `GitOpsParseFailed`; missing workload or
container matches, unsafe paths, and incompatible pod templates report
`GitOpsUpdateFailed`. Repeating the same desired update is a no-op and does not
create another Git commit.

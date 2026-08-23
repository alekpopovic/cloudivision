# Kustomize image releases

Use `kustomize-image` when the deployment repository manages image overrides in
a Kustomize `images` list. cloudivision changes only the selected list entry and
does not apply the rendered manifests to the cluster.

```yaml
spec:
  gitOps:
    enabled: true
    repoURL: https://github.com/acme/platform-gitops.git
    branch: main
    path: overlays/production/checkout
    strategy: kustomize-image
    environmentRef: production
    kustomize:
      kustomizationFile: kustomization.yaml
      imageName: checkout-placeholder
```

Given:

```yaml
images:
  - name: checkout-placeholder
    newName: ghcr.io/acme/checkout
    newTag: previous
  - name: metrics-sidecar
    newName: ghcr.io/acme/metrics
    newTag: stable
```

the first entry receives the Release image `newName` and either `newTag` or
`digest`; the sidecar remains unchanged. A digest removes a stale `newTag`.
`kustomizationFile` defaults to `kustomization.yaml`. If `imageName` is omitted,
the desired image repository is used as the Kustomize name and a missing entry
is appended for backward compatibility. If an explicit name is absent from the
file, the release fails with `GitOpsUpdateFailed`.

Repository-relative paths are validated against traversal. An update that is
already present returns the existing Git HEAD and creates no duplicate commit or
push.

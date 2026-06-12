# Supply-chain security hooks

cloudivision keeps Kubernetes CRD status as the source of truth for runtime build and release state. The v1alpha1 API now includes fields for supply-chain metadata so CI can record image digest, SBOM, scan, signature and provenance references without moving that state into a database.

## Implemented now

- `BuildRun.status.image.repository`, `tag` and `digest` record the produced image.
- `BuildRun.status.supplyChain` can record:
  - `sbomPath`
  - `sbomDigest`
  - `signatureRef`
  - `provenanceRef`
  - `scannerResultsRef`
- `PipelineTemplate.spec.supplyChain` declares requested hooks:
  - `generateSBOM`
  - `scanImage`
  - `signImage`
  - `requireSignedBaseImages`
- `Environment.spec.policy` declares release requirements:
  - `requireSignedImages`
  - `requireSBOM`
  - `blockCriticalVulnerabilities`
- The runner calls supply-chain hook interfaces after a successful image build.
- The Release controller blocks a release before any GitOps repository update when the target Environment policy is not satisfied.

## Noop in the MVP

The current hook implementations are intentionally noops:

- `NoopSBOMGenerator`
- `NoopScanner`
- `NoopSigner`
- `NoopProvenanceWriter`

They preserve existing behavior and return no external references. This means an Environment that requires signed images, an SBOM or scanner evidence will block releases until real integrations populate `BuildRun.status.supplyChain`.

## Future integrations

The hook interfaces are designed to be backed by concrete tools later:

- SBOM generation: Syft, Trivy or build-system native SBOM output.
- Vulnerability scanning: Grype, Trivy or registry-native scanner results.
- Image signing: Cosign keyless signing, KMS-backed keys or enterprise signing services.
- Provenance and attestations: SLSA provenance, in-toto attestations or OCI referrers.
- Base image verification: policy checks for signed base images before build execution.

Concrete integrations should keep secrets out of logs, write durable references into `BuildRun.status.supplyChain`, and avoid deploying directly from the CI runner. CD remains GitOps-driven.

## Release policy behavior

Release policy is evaluated before the Release controller updates the GitOps repository. If policy is not satisfied, the Release is marked `Failed` with reason `PolicyNotSatisfied`.

For example, a production Environment can require signed images and SBOM metadata:

```yaml
spec:
  type: production
  policy:
    requireSignedImages: true
    requireSBOM: true
    blockCriticalVulnerabilities: true
```

With the current noop hooks, that policy will block until a BuildRun has `status.supplyChain.signatureRef` and either `status.supplyChain.sbomPath` or `status.supplyChain.sbomDigest`. If critical vulnerability blocking is enabled, `status.supplyChain.scannerResultsRef` is also required.

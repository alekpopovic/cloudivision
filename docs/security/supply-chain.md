# Supply-chain security adapters

cloudivision stores artifact evidence in Kubernetes CRD status. CI records image,
SBOM, scan, signature, and provenance references; CD remains GitOps-driven and
evaluates Environment policy before changing desired state.

## Status and policy

`BuildRun.status.supplyChain` records `sbomPath`, `sbomDigest`, `signatureRef`,
`provenanceRef`, `scannerResultsRef`, and critical/high/medium/low vulnerability
counts. Successful hooks also set `SBOMGenerated`, `ImageScanned`, `ImageSigned`,
and `ProvenanceWritten` conditions.

An Environment can require signed images and SBOM evidence or block critical
vulnerabilities:

```yaml
spec:
  type: production
  policy:
    requireSignedImages: true
    requireSBOM: true
    blockCriticalVulnerabilities: true
```

Missing evidence moves the Release to `FailedValidation` with
`PolicyNotSatisfied`. A non-zero critical count uses
`CriticalVulnerabilitiesFound`. Evaluation happens before any GitOps write.

## Adapter configuration

Noop implementations remain the default, so upgrading does not invoke external
tools or signing identities. Select real adapters explicitly on a PipelineTemplate:

```yaml
spec:
  supplyChain:
    generateSBOM: true
    sbomAdapter: syft
    scanImage: true
    scannerAdapter: grype
    signImage: true
    signerAdapter: cosign
    cosignKeyless: false
    signingKeySecretRef:
      name: cosign-signing-key
      key: cosign.key
    provenanceAdapter: json
```

| Hook | Default | Opt-in adapter | Requirement |
| --- | --- | --- | --- |
| SBOM | `NoopSBOMGenerator` | `syft` | Syft generates SPDX JSON and the runner records its file SHA-256. |
| Scan | `NoopScanner` | `grype` | Grype writes JSON; the runner records its path and severity summary. |
| Sign | `NoopSigner` | `cosign` | Choose explicit keyless mode or a key Secret, never both. |
| Provenance | `NoopProvenanceWriter` | `json` | Built in; records BuildRun, source URL/SHA, template, image digest, timestamps, and evidence. |

The standard runner image intentionally does not bundle Syft, Grype, or Cosign.
Use a reviewed custom runner image containing pinned versions. If an opted-in tool
is absent, the BuildRun fails with `SBOMGenerationFailed`, `ImageScanFailed`, or
`ImageSigningFailed` and a clear missing-binary message. Optional/noop hooks do not
fail builds.

Cosign keyless mode is disabled unless `cosignKeyless: true`. Key-based mode
projects only the named Secret key read-only at
`/var/run/secrets/cloudivision-signing/key`; the runner cannot enumerate Secrets.
Tool output and key material are not logged.

## Limitations

SBOM, Grype, and JSON provenance paths currently refer to files in the runner
workspace. They are available while the Pod is retained but are not durable after
Job cleanup. Uploading them as OCI referrers or to an artifact store is future
work. JSON provenance is initial build metadata, not yet a signed SLSA/in-toto
attestation. Registry authentication is inherited from the runner environment and
must be scoped outside these adapters. `requireSignedBaseImages` remains a model
hook without a verification adapter.

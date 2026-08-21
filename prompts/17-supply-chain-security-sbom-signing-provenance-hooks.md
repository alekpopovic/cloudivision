# Prompt 17 — Supply Chain Security: SBOM, Signing, Provenance Hooks

Phase: Security

```text
Continue working on the cloudivision project.

Task:
Add design and initial implementation for supply-chain security hooks in cloudivision.

Goal:
Architecture must have room for image digest, SBOM, vulnerability scanning, image signing, provenance/attestations and policy that only signed images can be released to production.

Steps:
1. Extend CRD types if needed:
   - BuildRun.status.image repository/tag/digest.
   - BuildRun.status.supplyChain sbomPath, sbomDigest, signatureRef, provenanceRef, scannerResultsRef.
   - PipelineTemplate.spec.supplyChain generateSBOM, scanImage, signImage, requireSignedBaseImages.
   - Environment.spec.policy requireSignedImages, requireSBOM, blockCriticalVulnerabilities.
2. Add interfaces: SBOMGenerator, VulnerabilityScanner, ImageSigner, ProvenanceWriter.
3. Implement noop versions: NoopSBOMGenerator, NoopScanner, NoopSigner, NoopProvenanceWriter.
4. Runner calls hooks after image build and before Release creation/supply-chain completion.
5. If Environment requires a policy that is not satisfied, Release controller blocks release with reason PolicyNotSatisfied.
6. Document in /docs/security/supply-chain.md what exists, what is noop and future integration points.

Acceptance criteria:
- go test ./... passes.
- Noop hooks do not change existing behavior.
- Status models have fields for digest/SBOM/signature/provenance.
- Release policy can block production release if signature/SBOM is required but missing.
- Documentation clearly separates implemented from planned functionality.
```

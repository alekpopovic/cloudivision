# Prompt 30 — Real Supply Chain Adapters

Phase: Supply Chain

```text
Continue working on the cloudivision project.

Task:
Replace noop-only supply-chain hooks with real optional adapters.

Goal:
Add practical supply-chain functionality while keeping it optional and safe.

Adapters to add:
1. SBOMGenerator:
   - Keep NoopSBOMGenerator.
   - Add Syft-based adapter if syft binary is available.
   - If syft is missing, return clear error when SBOM generation is required.
   - Store SBOM file path and digest in BuildRun.status.supplyChain.
2. VulnerabilityScanner:
   - Keep NoopScanner.
   - Add Grype-based adapter if grype binary is available.
   - If scanner is missing and scan is required, fail with clear reason.
   - Store scanner result reference and severity summary if practical.
3. ImageSigner:
   - Keep NoopSigner.
   - Add Cosign-based adapter if cosign binary is available.
   - Support keyless mode only when explicitly configured.
   - Support key-based mode through Kubernetes Secret reference if model exists.
   - Store signatureRef.
4. ProvenanceWriter:
   - Keep NoopProvenanceWriter.
   - Add initial JSON provenance file writer with buildRun, repository URL, commit SHA, pipeline template, image digest, startedAt/completedAt.

Policy behavior:
- Environment requiring SBOM blocks Release when missing.
- Environment requiring signed images blocks Release when signatureRef missing.
- blockCriticalVulnerabilities=true blocks Release if scanner found critical vulnerabilities.

Runner behavior:
- Run hooks after image build/push.
- Update BuildRun conditions: SBOMGenerated, ImageScanned, ImageSigned, ProvenanceWritten.
- Do not log secrets.
- Do not fail optional hooks unless policy requires them.

Angular UI:
Add Supply Chain tab to BuildRun detail showing image digest, SBOM, scan, signature, provenance and policy decision.

Docs:
Update docs/security/supply-chain.md with adapters, required binaries, config, limitations and examples.

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Noop behavior remains default.
- Real adapters are opt-in.
- Missing required adapter fails with clear status reason.
- Release policy can block production release based on missing SBOM/signature or critical vulnerabilities.
```

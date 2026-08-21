# Prompt 39 — Public Release Process and v0.1.0

Phase: Release

```text
Continue working on the cloudivision project.

Task:
Create the public release process for cloudivision and prepare v0.1.0.

Goal:
Make releases repeatable, auditable and easy to consume.

Create:
- docs/development/release-process.md
- .github/workflows/release.yaml if GitHub Actions is used
- CHANGELOG.md if missing
- VERSION file if desired
- scripts/release/ or hack/release/ scripts

Release artifacts:
- controller image
- api image
- runner image
- web image
- Helm chart
- CRD bundle
- checksums
- SBOM if supply-chain tooling exists
- signatures if signing exists

Versioning rules:
- Use semver for application releases.
- Helm chart version must be updated.
- appVersion must match application version.
- Container images tagged with version, git SHA and optionally latest for convenience.
- CRD version notes must be included.

Release workflow:
1. Run checks: gofmt, go test ./..., go vet ./..., npm ci/build/test in /web, helm template, security-check, conformance if feasible.
2. Build images.
3. Push images.
4. Package Helm chart.
5. Generate checksums.
6. Generate SBOM if configured.
7. Sign artifacts if configured.
8. Create GitHub/Git release notes.
9. Publish Helm chart or package artifact.

CHANGELOG:
Add unreleased section and v0.1.0 template with Features, Fixes, Security, Breaking changes, Upgrade notes and Known issues.

Docs:
release-process.md explains cutting a release, testing RC, rollback, Helm chart update and artifact publishing.

Acceptance criteria:
- Release process is documented.
- v0.1.0 checklist exists.
- Release workflow exists or release scripts exist.
- CHANGELOG.md exists.
- Release artifacts are clearly defined.
- Current codebase can produce at least local release artifacts.
```

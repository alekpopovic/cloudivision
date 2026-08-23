# Release process

Application and chart releases use Semantic Versioning. `VERSION`, Helm
`version`, and Helm `appVersion` must match exactly. The served CRD remains
`v1alpha1`; every release note must state schema, default, validation, storage,
conversion and rollback effects even when the answer is “no change”.

## Artifacts

A public release contains controller, API, runner and web images; controller, API,
runner and CLI archives; the web static archive; Helm chart; CRD bundle;
`SHA256SUMS`; source SBOM when Syft is available; and signatures when configured.
Images are tagged with the semantic version and full Git SHA. Stable releases also
receive `latest`. GitHub Actions adds image SBOM/provenance attestations and signs
image digests keylessly with Cosign.

## Cut and test a release candidate

1. Start from a clean, protected `main` and choose `X.Y.Z-rc.N`.
2. Move relevant `Unreleased` entries in `CHANGELOG.md` into the version section.
3. Update `VERSION` plus Chart `version` and `appVersion`; add
   `docs/releases/vX.Y.Z.md` with install, artifact, compatibility, upgrade, and
   known-issue sections; document CRD changes.
4. Run the final gate, including live conformance and upgrade on a disposable kind
   cluster. Triage dependency and container scan findings.
5. Produce local evidence:

```sh
RELEASE_VERSION="$(tr -d '[:space:]' < VERSION)"
./scripts/release/build-local.sh "$RELEASE_VERSION"
sha256sum -c "dist/release/v${RELEASE_VERSION}/SHA256SUMS"
helm install cloudivision "dist/release/v${RELEASE_VERSION}/cloudivision-${RELEASE_VERSION}.tgz" \
  --namespace cloudivision --create-namespace --dry-run
```

To include local images without pushing:

```sh
RELEASE_BUILD_IMAGES=true ./scripts/release/build-local.sh "$RELEASE_VERSION"
```

Use `RELEASE_GENERATE_SBOM=true` to require Syft. Use a protected
`RELEASE_COSIGN_KEY` path to sign blobs. Never expose a key or passphrase in logs.
Conformance is opt-in because it targets a Kubernetes context:

```sh
RELEASE_RUN_CONFORMANCE=true ./scripts/release/build-local.sh "$RELEASE_VERSION"
```

## Publish

After RC installation, build, webhook, GitOps release, production approval,
rollback, upgrade and uninstall-retention tests pass:

```sh
git tag -s "v${RELEASE_VERSION}" -m "cloudivision v${RELEASE_VERSION}"
git push origin "v${RELEASE_VERSION}"
```

The tag workflow validates the version and matching release-notes file, reruns
checks, pushes four GHCR images, packages local artifacts, generates a source
SBOM, signs image digests and creates a GitHub Release using the reviewed notes.
Verify
the release is not published until all required jobs and environment approvals
pass. Download artifacts into a clean directory, verify checksums/signatures, and
install the packaged chart against the immutable version tags.

## Rollback and failed publication

Do not move or reuse a published tag. If the workflow fails before publication,
fix forward and rerun the same commit only when no conflicting artifact exists. If
artifacts were public, publish a patch release and mark the affected release in the
changelog.

For a workload rollback, use a compatible previous Helm revision or chart and
verify controller/API/runner image digests. Helm does not roll back CRDs: never
delete or blindly replace a CRD. Restore custom resources into an isolated cluster
when an older controller cannot read the new schema. Revoked images or signatures
must remain auditable in release notes.

## Release checklist

- [ ] Final gate recommends shipping; critical/high dependency findings triaged.
- [ ] `VERSION`, chart version, appVersion, OpenAPI metadata, release notes,
      changelog, and tag all name the same version.
- [ ] `gofmt`, Go test/vet, npm ci/build/test, Helm and security checks pass.
- [ ] Clean-cluster conformance, upgrade and limited scale evidence reviewed.
- [ ] RC verifies first build, signed webhook, GitOps release and approval.
- [ ] CRD compatibility and upgrade notes reviewed; backup restore exercised.
- [ ] Four images have version/SHA tags, SBOM, provenance and signatures.
- [ ] Archives, chart, CRD bundle and checksum verification pass.
- [ ] GitHub release notes match `CHANGELOG.md`; rollback owner is assigned.

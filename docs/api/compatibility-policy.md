# CRD compatibility policy

This policy applies to the `cicd.cloudivision.io` API group and begins with cloudivision v0.1. Release notes may provide stronger guarantees, but must not silently weaken this policy.

## API maturity

### v1alpha1

`v1alpha1` is experimental. Fields and behavior may change between minor releases, including breaking changes, when the change fixes an unsafe model or unlocks the beta design. Even in alpha, cloudivision will:

- preserve stored objects across patch releases;
- avoid removing a field without a documented migration path;
- publish upgrade notes for schema, default or behavior changes;
- keep status additive where practical;
- never reuse an old field name with a different meaning;
- test the last released alpha to current-alpha upgrade before a release.

Alpha users should pin the Helm chart/controller version, keep manifests in source control and review every minor-release upgrade note.

### v1beta1

Before serving `v1beta1`, cloudivision must have:

- a reviewed resource/field model and structural OpenAPI schemas;
- defaulting and CEL validation tests against a real API server;
- a conversion strategy for every simultaneously served version;
- round-trip tests that preserve all representable user intent;
- an upgrade test from the last supported `v1alpha1` release;
- documented deprecation periods and storage-version migration steps;
- at least one release cycle of conformance evidence.

Beta fields are expected to remain source compatible. Breaking changes require a new served version, conversion and release-note migration instructions. A beta field can be removed only after at least two minor releases of deprecation unless retaining it creates an active security or data-loss risk.

### v1

`v1` is the stable contract. Before promotion, the API needs sustained beta usage, upgrade/conversion coverage, documented condition semantics, stable defaulting and validation, and compatibility review by maintainers. Stable fields are not removed or semantically repurposed within v1. New optional fields and enum values may be added when clients are expected to tolerate them.

## Compatibility rules

### Fields and wire format

- JSON field names are the compatibility boundary, not Go member names.
- Additive optional fields are preferred.
- Required fields may not be added to an existing stable version unless admission supplies a safe default for old objects.
- Enum expansion is treated as a client compatibility event; consumers must handle unknown values before expansion.
- Lists must document whether order is semantic and which key, if any, identifies an entry.
- Status is controller-owned. Users and clients must not depend on undocumented condition ordering.

### Deprecation

Deprecated fields remain readable and round-trip during their support window. Go comments and generated OpenAPI descriptions use `Deprecated:` and release notes state:

1. deprecated field and replacement;
2. first deprecated release;
3. earliest removal version;
4. conversion/default interaction;
5. manifest migration example.

Controllers should prefer the replacement when both fields exist and surface a warning condition or admission warning when practical. Conflicting old/new values must be rejected or resolved by a documented deterministic rule.

### Defaulting

Defaults are API behavior. Defaults should be applied by CRD schema or a versioned defaulting webhook, not only inside reconcilers. A new default must not reinterpret a value that was previously stored explicitly. Conversion converts representation; it must not apply business defaults. Defaulting tests cover create, read-after-create, conversion and old stored objects.

### Conversion and storage

When more than one version is served, cloudivision will use a hub-and-spoke model. The current storage/hub candidate is `v1alpha1` only until `v1beta1` exists; the intended beta migration makes `v1beta1` the storage hub while `v1alpha1` remains a spoke for its support window.

Every spoke-to-hub and hub-to-spoke conversion must:

- be deterministic and side-effect free;
- preserve unknown or non-representable data through an explicitly reviewed mechanism, or reject the conversion before data loss;
- preserve metadata and status semantics;
- have round-trip and fuzz/property coverage;
- avoid contacting external systems.

A conversion webhook is not added to CRD `conversion.strategy` until its Deployment, Service, TLS/certificate rotation, readiness and failure-mode tests are production-ready. The package under `internal/conversion` is intentionally an in-process design skeleton and does not alter installs.

### Release and upgrade notes

Every release that changes CRDs includes an `API and CRD changes` section containing schema/default/CEL changes, deprecations, conversion/storage changes, required manual steps, rollback constraints and a link to tested upgrade paths. Generated CRDs are committed and CI must fail on generation drift.

Upgrade tests install the previous supported CRDs/controller, create representative objects in every lifecycle phase, upgrade CRDs/controller, then verify objects remain readable, convertible and reconcilable without duplicate child resources.

## Exceptions

An emergency security or data-loss fix may shorten an alpha/beta deprecation window. Maintainers must document the risk, affected releases, detection query, backup requirement, exact migration and rollback limits. Stable v1 wire compatibility is not waived; a new API version or admission rejection must be used.

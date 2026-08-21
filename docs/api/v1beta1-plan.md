# v1beta1 API plan

This is a proposal, not a served API. No `v1beta1` CRD or conversion webhook is installed today.

## Design goals

- Make namespace/reference semantics explicit.
- Separate reusable pipeline intent from executor-specific configuration.
- Make artifact identity digest-first and supply-chain policy strongly typed.
- Move approval decisions out of mutable desired-state fields.
- Ensure every invalid cross-field combination is rejected at admission.
- Preserve all supported v1alpha1 objects through round-trip conversion.

## Proposed changes

| Resource | v1alpha1 issue | v1beta1 direction |
|---|---|---|
| Project | `namespace` and object namespace can diverge; isolation booleans are weakly defaulted | Rename to `workloadNamespace`; strongly default isolation; define whether cross-namespace targeting is allowed. |
| Repository | provider details and webhook are monolithic; disabled webhook still serializes an empty reference | Split provider/webhook config into discriminated structures; make webhook pointer/optional and require secret only when enabled. |
| PipelineTemplate | Job executor ignores `step.image`; builder/security capability is ambiguous | Define workspace and step-container semantics; split `execution`, `imageBuild`, `resources`, `securityPolicy`; use keyed-list step names with uniqueness validation. |
| BuildRun | mutable-looking execution intent and free-form params; image tag can be ambiguous | Declare immutable fields; type parameter values; split source revision and artifact output; require GitOps target only when promotion is enabled. |
| Environment | `namespace` name is ambiguous; provider status is free text | Rename to `targetNamespace`; use a typed provider union; strongly type sync/health; make production approval opt-out an explicit policy-controlled exception. |
| Release | approval decision lives in spec; image can lack tag/digest; `environmentRef` optional | Move approval decisions to an Approval subresource/resource or append-only action; require environment; use digest-first ArtifactRef and typed promotion status. |

## Fields to remove or replace

- Replace `PipelineTemplate.spec.security.allowPrivileged` with explicit capability policy; no implicit privileged path.
- Replace `BuildRun.spec.commitSHA` plus `revision` ambiguity with a typed `source.revision` containing requested and resolved revisions.
- Replace free-form `Release.status.deployment.provider` and status strings with enums that still tolerate unknown provider extensions.
- Review `Project.spec.defaultRegistry`; prefer a typed registry reference that can identify credential/policy configuration.
- Move mutable approval actor/timestamps out of Release spec while preserving v1alpha1 reads during conversion.

No field is removed until conversion can preserve it or admission rejects the unsupported case with migration guidance.

## Defaults and validation before beta

- Defaults must be identical between generated OpenAPI, Go/API clients and conversion tests.
- CEL enforces enabled webhook/GitOps dependencies, production approval and supply-chain policy prerequisites.
- Reference names use DNS-compatible validation.
- quantities and timeouts have upper/lower bounds.
- steps are a map-list keyed by name and require at least one step or enabled image build.
- BuildRun source/execution intent becomes immutable after creation.
- Release artifact requires a digest for production; tag-only may remain valid in development.

## Conversion rollout

1. Inventory real v1alpha1 objects and edge cases from conformance/dogfood clusters.
2. Implement pure v1alpha1 ↔ internal-hub ↔ v1beta1 functions under `internal/conversion`.
3. Add round-trip, lossy-case and fuzz tests before exposing a second version.
4. Ship optional webhook Deployment/Service/TLS behind an off-by-default Helm value and test failure/rotation behavior.
5. Serve `v1beta1` without making it storage; verify conversion in upgrade tests.
6. Switch storage to `v1beta1`, run storage-version migration and verify `status.storedVersions`.
7. Deprecate served v1alpha1 only after the policy support window and published migration tooling.

## Open decisions

- Whether Project may target a namespace other than its own.
- Job executor model for true per-step images and shared workspace.
- Separate Artifact and Approval resources versus embedded typed structures.
- Unknown-field preservation approach for alpha fields with no beta representation.
- Certificate management dependency versus a self-managed webhook certificate path.

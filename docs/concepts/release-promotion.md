# Release promotion

Promotion creates a new immutable Release for another Environment. Call `POST /api/v1/releases/{namespace}/{name}/promote` with `targetEnvironmentRef` and an optional `actor`. The new Release reuses the source image digest (falling back to the source BuildRun's recorded image) and records `spec.promotedFrom`.

The target Environment owns approval and supply-chain policy. Production targets require approval by default, so their new Release remains awaiting approval until an authorized actor approves it. Promotion never changes the source Release or directly applies manifests; the normal GitOps controller path performs delivery.

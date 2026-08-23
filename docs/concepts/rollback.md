# Release rollback

Rollback is a forward-moving, auditable Release rather than a mutation of history. Call `POST /api/v1/releases/{namespace}/{name}/rollback` with a previously deployed `targetReleaseRef`, optional `actor`, and `reason`.

The created Release restores the previous immutable image, records `rollbackOf` and `rollbackTo`, and follows the same approval and GitOps reconciliation as any other Release. When it converges, its phase is `RolledBack`. Environment policy can block an image without a digest or signature. The original and previous Releases remain unchanged.

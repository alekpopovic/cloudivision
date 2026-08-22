# Disaster recovery

Start by stopping automation that can amplify damage: suspend external webhook delivery, GitOps auto-sync where appropriate, and new release approvals. Capture Events, pod logs, CR YAML, and external-system status before restarting components.

## Controller or API unavailable

Desired state remains in CRs while controllers are down. Check Deployment availability, probes, RBAC denials, configuration, and leader election. Restore the matching chart version and confirm one active controller before resuming webhooks. The API can be restored independently; Kubernetes-native reconciliation does not depend on its in-memory state.

## Runner Job stuck

Inspect the Job, Pod, Events, image pulls, quotas, ServiceAccount, NetworkPolicy, and deadline. Do not create a second Job manually. After correcting the dependency, delete the failed or stuck Job only when the BuildRun controller can safely recreate it or the run will be explicitly retried as a new BuildRun. Preserve logs first.

## GitOps repository unavailable

Pause promotions and keep Releases visible. Confirm DNS, credentials, provider rate limits, and repository integrity. Never bypass the GitOps repository by applying application manifests directly from the CI runner. Reconcile the same Release after service recovery and verify commit or PR idempotency.

## Registry unavailable

Pause builds and promotions, retain local evidence and digests, and verify registry authentication, quota, replication, and immutable tag policy. Do not promote a tag whose digest cannot be resolved. Recover from a replicated registry or rebuild only from a verified commit with new provenance.

## PostgreSQL unavailable

The optional audit/cache database outage must not become the only loss of runtime state; CR status remains authoritative. Restore from a consistent snapshot, verify migrations, audit timestamps, and webhook idempotency before enabling the API. Keep failed audit writes visible rather than silently discarding them.

## Cluster loss

Provision a new cluster, restore CRDs and external dependencies, then follow `backup-restore.md`. Keep GitOps auto-sync controlled until target namespaces and credentials are verified. Compare resource counts, sample specs/status, GitOps commits, registry digests, and audit continuity. Finish with a non-production BuildRun and Release before reopening production automation.

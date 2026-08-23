# Flux status

Set an Environment's GitOps provider to `flux`, then set `applicationName`, `namespace`, and `resourceKind` (`Kustomization` or `HelmRelease`). `Kustomization` is the compatibility default.

The controller reads the selected Flux object as unstructured data. Its Ready condition maps to separate Release sync and health fields, while `lastAppliedRevision` (or `lastAttemptedRevision`) and the Ready transition time are retained. Ready=True maps to Synced/Healthy; Ready=False maps to OutOfSync/Degraded.

Missing Flux CRDs produce `ProviderUnavailable`; missing configured resources produce `DeploymentResourceMissing`. The chart grants only get/list/watch for Kustomizations and HelmReleases.

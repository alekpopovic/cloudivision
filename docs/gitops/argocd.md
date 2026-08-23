# Argo CD status

Set an Environment's GitOps provider to `argocd`, with `applicationName` and the namespace containing the Application. The controller reads the Application as an unstructured resource, so cloudivision does not require the Argo CD Go API dependency.

Release deployment status preserves Argo CD sync and health independently and also records the operation phase, observed revision and `reconciledAt`. A `Synced` and `Healthy` Application completes the Release. A missing CRD produces `ProviderUnavailable`; a missing Application produces `DeploymentResourceMissing`. Both are observable and retried without crashing reconciliation.

Grant the controller read-only access to `argoproj.io/applications`. The Helm chart includes this permission.

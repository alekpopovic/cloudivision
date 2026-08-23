# v1beta1 API

`v1beta1` is an experimental Go API shape. It is **not served by the installed
CRDs and is not the storage version**. Existing clusters continue to use
`cicd.cloudivision.io/v1alpha1`.

The beta model makes workload and target namespaces explicit, groups pipeline
execution/image-build/security settings, gives BuildRun source revisions and
parameters typed structures, and renames release images to digest-first
artifacts. Repository webhook and credential blocks are optional pointers.

Pure conversions live in `internal/conversion`. Round-trip tests cover all six
resources. Alpha Release approval decisions have no final beta equivalent;
conversion temporarily carries them in the deprecated
`legacyApprovalDecision` field so no user intent is silently lost. New beta
clients must not write that field.

Before this version can be served, the project must complete conversion
webhook, upgrade, unknown-field and storage migration testing described in
[`v1beta1-plan.md`](v1beta1-plan.md).

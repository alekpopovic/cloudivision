# Controllers

The controller manager reconciles `Project`, `BuildRun`, and `Release` resources.
Reconcilers must be idempotent: repeated calls should converge on the same child
resources and status. Build execution is selected through the executor interface;
the domain model does not depend on Kubernetes Job or Tekton details.

Key rules:

- record user-visible phase, conditions and references in CRD status;
- use ownership labels and controller references for child resources;
- do not log Secret data;
- set workload requests, limits, deadlines and restricted security contexts;
- requeue transient failures and make terminal failures explicit;
- never deploy application manifests directly from the CI runner.

Run controller tests with:

```sh
go test ./internal/controller/... ./internal/executor/...
go test ./...
go vet ./...
```

For a focused test:

```sh
go test ./internal/controller -run TestBuildRunReconciler -count=1
```

When adding fields, update API types, generated CRDs, controller behavior, tests,
examples and [CRD reference](../reference/crds.md) together.

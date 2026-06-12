# Testing strategy

cloudivision uses a layered test strategy so contributors can get useful feedback without a Kubernetes cluster, while still leaving a path for kind/envtest coverage.

## Unit tests

Run:

```sh
make test-unit
```

This covers the core packages under `api`, `internal/domain`, `internal/webhook`, `internal/build`, `internal/gitops`, `internal/executor`, `internal/audit`, `internal/auth`, `internal/redact` and `internal/runner`.

## Controller tests

Run:

```sh
make test-controller
```

Controller tests currently use controller-runtime fake clients for fast reconciliation coverage. The target prints a clear notice when `KUBEBUILDER_ASSETS` is not set, so envtest coverage is not confused with fake-client coverage. Future envtest suites should live in `internal/controller` and require `KUBEBUILDER_ASSETS`.

Covered behavior includes:

- Project namespace, ServiceAccount, Role, RoleBinding and NetworkPolicy reconciliation.
- BuildRun Job creation and idempotency.
- Job success/failure status propagation.
- BuildRun success creating one Release when GitOps is enabled.
- Release approval and deployment status flows.

## API tests

Run:

```sh
make test-api
```

API tests use `httptest` and fake Kubernetes clients. They cover BuildRun creation/listing, webhook signatures, logs errors, JSON errors, auth disabled mode and local UI CORS.

## Web tests

Run:

```sh
make test-web
```

Angular tests cover app shell rendering, dashboard smoke rendering, status badge labels/classes, backend error display including request ID, BuildRun list API calls and manual BuildRun form validation.

## E2E skeleton

Run:

```sh
make test-e2e
```

The current e2e target checks for `kind`, `kubectl`, `helm` and `docker`. It exits with a clear error when prerequisites are missing. A full e2e should build/load images, install cloudivision with Helm, apply `deploy/examples`, wait for a BuildRun and assert logs/status.

## Full local suite

Run:

```sh
make test-all
```

This runs formatting, unit/controller/API/web tests, `go vet`, builds the Go binaries and Angular UI, and renders the Helm chart.

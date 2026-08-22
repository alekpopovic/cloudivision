# Testing

Use the smallest focused suite while developing, then run the complete gates:

```sh
gofmt -w ./api ./cmd ./internal
go test ./...
go vet ./...
npm --prefix web ci
npm --prefix web run build
npm --prefix web test -- --watch=false --browsers=ChromeHeadless
helm template cloudivision charts/cloudivision --include-crds >/tmp/cloudivision.yaml
```

Additional suites exercise cluster behavior:

```sh
make security-check
make conformance
make upgrade-test
make scale-test
```

Conformance and upgrade tests require a disposable Kubernetes cluster and the
documented environment variables. They may create and delete namespaces. Read the
corresponding scripts before pointing them at any shared cluster. PostgreSQL audit
tests run only when `CLOU_DIVISION_TEST_DATABASE_URL` is set.

Tests should cover domain transitions, repeat reconciliation, malformed API input,
authorization, policy denials and redaction. A behavior change is incomplete until
its user-visible documentation and negative-path test are updated.

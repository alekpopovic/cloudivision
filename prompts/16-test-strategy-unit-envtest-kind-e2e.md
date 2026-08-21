# Prompt 16 — Test Strategy: Unit, Envtest, kind E2E

Phase: Testing

```text
Continue working on the cloudivision project.

Task:
Set up a serious test strategy for cloudivision.

Unit tests:
- /internal/domain
- /internal/webhook
- /internal/build
- /internal/gitops
- /internal/executor
- /internal/audit
- /internal/auth

Controller tests:
- Use envtest if available.
- Test Project controller namespace/SA/RBAC creation.
- Test BuildRun controller creates Job and does not duplicate Job.
- Test Job success/failure updates BuildRun.
- Test BuildRun Succeeded + gitOps enabled creates Release.
- Test Release awaiting approval does not push GitOps change.
- Test Release deploy updates status.

API tests:
- Use httptest for create/list BuildRuns, valid/invalid webhook signature, logs pod not found, JSON error format, auth disabled mode and CORS config.

Angular tests:
- App shell renders.
- Dashboard page renders.
- Status badge renders correct classes/labels.
- API error component displays code/message/request ID.
- BuildRun list component calls service.
- Manual BuildRun form validates required fields.

E2E:
- Add /test/e2e with kind-oriented test skeleton.
- Full e2e may be incomplete initially.
- Add make target e2e that checks prerequisites and gives clear error if kind is not installed.

Makefile targets:
- test-unit
- test-controller
- test-api
- test-web
- test-e2e
- test-all

CI:
- Add GitHub Actions workflow for gofmt, go test ./..., go vet ./..., npm ci/build/test in /web, and helm template with caches.

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- make test-unit works.
- envtest/kind absence gives clear skip/error, not false green.
- GitHub Actions workflow exists and uses cache for Go and Node.
```

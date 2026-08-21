# Prompt 01 — Repository Scaffold

Phase: Foundation

```text
Continue working on the cloudivision project.

Task:
Create the initial monorepo scaffold for a Kubernetes-native CI/CD platform.

Technology decisions:
- Go for the operator, API server and runner.
- Angular + TypeScript + Tailwind CSS for the frontend.
- Kubebuilder-compatible layout for the Kubernetes operator.
- Helm chart for installation.
- PostgreSQL is planned, but does not need to run in the first scaffold.
- The MVP executor uses Kubernetes Jobs.
- Tekton support will be added later through an executor interface.

Steps:
1. Inspect the existing repository contents.
2. Initialize a Go module if one does not exist:
   - module name: github.com/cloudivision/cloudivision
3. Create the following directory structure:
   - /api/v1alpha1
   - /cmd/controller
   - /cmd/api
   - /cmd/runner
   - /internal/controller
   - /internal/domain
   - /internal/executor
   - /internal/build
   - /internal/git
   - /internal/gitops
   - /internal/webhook
   - /internal/auth
   - /internal/audit
   - /internal/kube
   - /web
   - /charts/cloudivision
   - /config
   - /deploy/examples
   - /docs
   - /docs/adr
   - /hack
   - /test
4. Add README.md explaining what cloudivision is, architecture, MVP flow, and early development status.
5. Add Makefile targets: fmt, test, vet, lint, build, run-api, run-controller, run-runner, docker-build-api, docker-build-controller, docker-build-runner, docker-build-web, manifests, generate, helm-template.
6. Add minimal Go entrypoints:
   - /cmd/api/main.go with GET /healthz and GET /readyz.
   - /cmd/controller/main.go placeholder that starts and logs controller is not implemented yet.
   - /cmd/runner/main.go placeholder that reads BUILD_RUN_NAME and logs startup.
7. Add initial Angular scaffold in /web:
   - minimal Angular app
   - Tailwind CSS configured
   - simple landing page saying “cloudivision”
   - npm scripts: start, build, test, lint if configured
8. Add .gitignore for Go, Node, Angular, Kubernetes local files, editor and OS files.
9. Add docs/adr/0001-kubernetes-native-architecture.md documenting CRDs as source of truth, Jobs as first executor, Tekton as future adapter, GitOps CD, and Angular + Tailwind frontend.

Acceptance criteria:
- go test ./... passes.
- Makefile target fmt works.
- README and ADR exist.
- Go entrypoints exist and build.
- Angular app builds with npm run build.
- No hardcoded secrets.
- No Docker-in-Docker or privileged pod assumptions.
```

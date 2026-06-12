# Initial monorepo scaffold

User prompt:

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

/api/v1alpha1
/cmd/controller
/cmd/api
/cmd/runner
/internal/controller
/internal/domain
/internal/executor
/internal/build
/internal/git
/internal/gitops
/internal/webhook
/internal/auth
/internal/audit
/internal/kube
/web
/charts/cloudivision
/config
/deploy/examples
/docs
/docs/adr
/hack
/test

4. Add a basic README.md explaining:
   - what cloudivision is
   - the high-level architecture
   - the MVP flow:
     webhook -> API -> BuildRun CR -> controller -> Kubernetes Job -> registry -> GitOps repo -> Argo CD/Flux
   - project status: early development

5. Add a Makefile with these targets:
   - fmt
   - test
   - vet
   - lint
   - build
   - run-api
   - run-controller
   - run-runner
   - docker-build-api
   - docker-build-controller
   - docker-build-runner
   - docker-build-web
   - manifests
   - generate
   - helm-template

6. Add minimal Go entrypoints:
   - /cmd/api/main.go:
     - HTTP server
     - GET /healthz
     - GET /readyz
   - /cmd/controller/main.go:
     - placeholder process that starts and logs that the controller is not implemented yet
   - /cmd/runner/main.go:
     - placeholder CLI
     - reads BUILD_RUN_NAME
     - logs startup

7. Add an initial Angular scaffold in /web:
   - create a minimal Angular app
   - configure Tailwind CSS
   - add a simple landing page that says “cloudivision”
   - add npm scripts:
     - start
     - build
     - test
     - lint if configured

8. Add a basic .gitignore for:
   - Go build output
   - Node dependencies
   - Angular output
   - Kubernetes local files
   - editor and OS files

9. Add docs/adr/0001-kubernetes-native-architecture.md with the decision:
   - cloudivision uses Kubernetes CRDs as the source of truth for runtime state
   - build execution starts with Kubernetes Jobs
   - Tekton is a future executor adapter
   - CD goes through GitOps, not direct apply from the CI runner
   - the frontend uses Angular + Tailwind CSS

Acceptance criteria:
- go test ./... passes.
- Makefile target fmt works.
- README and ADR exist.
- Go entrypoints exist and build.
- Angular app builds with npm run build.
- No hardcoded secrets.
- No Docker-in-Docker or privileged pod assumptions.
```

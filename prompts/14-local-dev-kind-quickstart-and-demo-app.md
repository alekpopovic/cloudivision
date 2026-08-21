# Prompt 14 — Local Dev: kind Quickstart and Demo App

Phase: Installation

```text
Continue working on the cloudivision project.

Task:
Add a local developer quickstart for running cloudivision in a kind cluster.

Steps:
1. Add /hack/kind-create.sh that creates kind cluster cloudivision-dev and enables a local registry if practical.
2. Add /hack/kind-load-images.sh that builds controller/api/runner/web images and loads them into kind.
3. Add /hack/install-dev.sh that installs CRDs and deploys controller/API/web through Helm or kustomize.
4. Add /deploy/examples: project.yaml, repository.yaml, pipeline-template-nodejs.yaml, buildrun-manual.yaml, environment-dev.yaml.
5. Add /deploy/demo-app: minimal Node.js or Go app, Dockerfile and test command.
6. Update README quickstart: prerequisites, create kind cluster, build images, install cloudivision, apply sample CRs, check BuildRun, view logs, open Angular web UI.
7. Add troubleshooting for CRDs, runner image pull, BuildKit, RBAC, Pod Security, Git credentials, Angular API connectivity and CORS.

Acceptance criteria:
- Scripts are idempotent where practical.
- Scripts use set -euo pipefail.
- README has concrete commands.
- Demo app can trigger a BuildRun without external Git if practical, or docs clearly explain local/public repo usage.
- Angular UI quickstart is documented.
```

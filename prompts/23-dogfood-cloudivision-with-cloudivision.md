# Prompt 23 — Dogfood cloudivision With cloudivision

Phase: Dogfood

```text
Continue working on the cloudivision project.

Task:
Create a dogfood setup where cloudivision can build and test the cloudivision repository itself.

Goal:
Use cloudivision as its own CI/CD platform. This should expose real product issues in runner behavior, logs, image builds, Angular builds, Helm validation and GitOps flows.

Create:
- /deploy/dogfood/project.yaml
- /deploy/dogfood/repository.yaml
- /deploy/dogfood/pipeline-template-go.yaml
- /deploy/dogfood/pipeline-template-web.yaml
- /deploy/dogfood/pipeline-template-images.yaml
- /deploy/dogfood/pipeline-template-release.yaml
- /deploy/dogfood/environment-dev.yaml
- /docs/dogfood.md

Dogfood pipeline requirements:
PipelineTemplate: cloudivision-go
- gofmt check
- go test ./...
- go vet ./...
- build controller binary
- build api binary
- build runner binary

PipelineTemplate: cloudivision-web
- npm ci in /web
- npm run build in /web
- npm test in /web if configured

PipelineTemplate: cloudivision-images
- build controller image
- build api image
- build runner image
- build web image
- optionally push to configured registry

PipelineTemplate: cloudivision-release
- helm template charts/cloudivision
- validate rendered YAML does not contain privileged: true, docker.sock or hostPath unless explicitly allowed
- optionally create GitOps Release to dev

Documentation:
docs/dogfood.md explains dogfooding, Repository URL configuration, registry credentials, manual trigger, status/log inspection and known limitations.

Acceptance criteria:
- Dogfood manifests exist.
- At least one dogfood BuildRun can run without requiring external paid services.
- Dogfood docs include exact kubectl/API commands.
- Dogfood pipeline validates both Go backend and Angular frontend.
- No dogfood manifest grants cluster-admin to the runner.
```

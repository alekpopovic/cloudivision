# 66. Admission Webhooks and Defaulting

```text
Continue working on the cloudivision project.

Task:
Add admission webhook support for defaulting and validation.

Goal:
Move advanced defaulting/validation out of controllers where appropriate.

Webhooks:
- mutating defaulting webhook
- validating webhook

Resources:
- Project
- Repository
- PipelineTemplate
- BuildRun
- Environment
- Release

Defaulting examples:
- Project.defaultBranch=main
- PipelineTemplate build defaults
- BuildRun executor=job
- Environment production requires approval default

Validation examples:
- no privileged pipeline unless explicitly allowed
- gitOps strategy required if gitOps enabled
- Repository webhook secret required if webhook enabled
- Release production requires digest if policy requires it

Helm:
- Webhooks disabled by default unless cert-manager or cert generation is configured.
- Add docs for enabling webhooks.
- Do not break simple install.

Tests:
- webhook unit tests
- envtest webhook tests if practical

Docs:
- docs/operations/admission-webhooks.md

Acceptance criteria:
- go test ./... passes.
- Webhook code exists.
- Webhooks are optional.
- Default install remains working.
- Validation errors are clear.
```

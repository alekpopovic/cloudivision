# Prompt 25 — CRD Validation Hardening With Defaults and CEL

Phase: API Evolution

```text
Continue working on the cloudivision project.

Task:
Harden cloudivision CRD validation and defaulting.

Goal:
Prevent invalid resources from being accepted by Kubernetes when the error can be detected at admission time.

Review and improve validation for Project, Repository, PipelineTemplate, BuildRun, Environment and Release.

Add or verify:
1. Enum validation for BuildRun executor, Repository provider, PipelineTemplate build builder, Environment type and Release phase/status values where appropriate.
2. Required fields:
   - Project.spec.namespace
   - Repository.spec.url and projectRef
   - PipelineTemplate.spec.steps where required
   - BuildRun projectRef/repositoryRef/pipelineTemplateRef
   - Release projectRef/environmentRef/buildRunRef
3. Defaults:
   - Project.spec.defaultBranch = main
   - PipelineTemplate.spec.build.contextDir = "."
   - PipelineTemplate.spec.build.dockerfile = "Dockerfile"
   - PipelineTemplate.spec.build.push = true
   - PipelineTemplate.spec.security.runAsNonRoot = true
   - BuildRun.spec.executor = job
4. Structural validation:
   - PipelineTemplate step names non-empty and unique if supported
   - BuildRun gitOps strategy required when gitOps.enabled=true
   - Repository webhook secretRef required when webhook.enabled=true
   - Environment production should default/require approval where practical
   - Release image tag or digest present
5. CEL validation with x-kubernetes-validations where appropriate:
   - gitOps.enabled implies gitOps.strategy set
   - webhook.enabled implies webhook.secretRef.name set
   - production requires approval unless explicitly allowed
   - supply-chain policy requirements imply required fields
6. Update docs/api/v1alpha1.md with validation/default notes.

Acceptance criteria:
- CRD YAML contains stronger schema validation.
- Invalid sample resources are rejected in tests where feasible.
- Valid sample resources still apply.
- go test ./... passes.
- make manifests passes if controller-gen is available.
- docs/api/v1alpha1.md is updated.
```

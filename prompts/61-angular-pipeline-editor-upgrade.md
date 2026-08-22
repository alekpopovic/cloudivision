# 61. Angular Pipeline Editor Upgrade

```text
Continue working on the cloudivision project.

Task:
Upgrade the Angular PipelineTemplate editor.

Goal:
Users should be able to create and edit PipelineTemplates without manually writing YAML.

Features:
1. Visual editor
- Add step
- Remove step
- Reorder steps
- Edit image/command/args/workingDir/env
- Continue-on-error toggle
- Timeout field

2. Build settings
- enable build
- contextDir
- Dockerfile
- builder
- image repository
- push toggle
- build args

3. YAML mode
- Show YAML representation
- Allow edit if practical
- Validate before save

4. Validation
- step name required
- image required
- command required
- duplicate step names warned
- unsafe privileged option warning

5. Catalog integration
- Start from built-in template.

Tests:
- create step
- remove step
- validation errors
- save template
- catalog import

Acceptance criteria:
- npm run build passes.
- npm test passes if configured.
- Pipeline editor works with backend DTOs.
- UI prevents obvious invalid templates.
- Backend validation remains source of truth.
```

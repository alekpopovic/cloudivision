# 65. CRD v1beta1 API Implementation

```text
Continue working on the cloudivision project.

Task:
Start implementing v1beta1 API types.

Goal:
Prepare cloudivision for a more stable public API.

Steps:
1. Create /api/v1beta1.
2. Define v1beta1 versions of:
- Project
- Repository
- PipelineTemplate
- BuildRun
- Environment
- Release

3. Improve naming and structure based on docs/api/v1beta1-plan.md.
4. Add conversion functions between v1alpha1 and v1beta1.
5. Do not make v1beta1 storage version unless fully ready.
6. Add tests for conversion round trips.
7. Add docs/api/v1beta1.md.

Rules:
- Preserve user intent during conversion.
- Do not silently drop important fields.
- Clearly document fields that cannot convert perfectly.
- Keep current installs working.

Acceptance criteria:
- go test ./... passes.
- v1beta1 types compile.
- Conversion tests exist.
- v1alpha1 remains supported.
- v1beta1 docs exist.
```

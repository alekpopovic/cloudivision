# Prompt 24 — CRD Compatibility and Versioning Plan

Phase: API Evolution

```text
Continue working on the cloudivision project.

Task:
Create a CRD compatibility and versioning plan for cloudivision.

Context:
Current APIs are v1alpha1. Before public releases, cloudivision needs clear rules for evolving CRD schemas without surprising users.

Goal:
Define how cloudivision will evolve from v1alpha1 to v1beta1 and eventually v1.

Deliverables:
1. docs/api/compatibility-policy.md
2. docs/api/v1alpha1.md
3. docs/api/v1beta1-plan.md
4. Initial conversion webhook skeleton if reasonable
5. Upgrade test skeleton

Policy should define:
- What v1alpha1 means.
- What breaking changes are allowed in v1alpha1.
- What will be required before v1beta1.
- What will be required before v1.
- How deprecated fields are marked.
- How defaulting is handled.
- How conversion will be tested.
- How CRD upgrade notes are documented per release.

docs/api/v1alpha1.md should document Project, Repository, PipelineTemplate, BuildRun, Environment, Release, important spec/status fields, lifecycle phases, conditions and examples.

docs/api/v1beta1-plan.md should propose fields to rename, split, strongly type, validate, default or remove before v1beta1.

Conversion webhook skeleton:
- Add package structure for future conversion logic if not already present.
- Add comments explaining hub/spoke approach if used.
- Do not introduce a broken required webhook into Helm install.
- Conversion support must be optional/skeleton unless fully working.

Upgrade test skeleton:
- Create test/upgrade or internal tests that can eventually install old CRDs, create v1alpha1 objects, upgrade controller/CRDs and assert objects still reconcile.

Acceptance criteria:
- Compatibility policy exists.
- v1alpha1 API is documented.
- v1beta1 plan exists.
- Conversion webhook skeleton does not break current installs.
- Upgrade test plan or skeleton exists.
- go test ./... passes.
```

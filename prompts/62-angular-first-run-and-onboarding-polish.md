# 62. Angular First-Run and Onboarding Polish

```text
Continue working on the cloudivision project.

Task:
Polish the Angular first-run experience.

Goal:
A new user should be guided from empty installation to first successful BuildRun.

First-run wizard:
1. Welcome
2. Create Project
3. Add Repository
4. Select PipelineTemplate from catalog
5. Configure webhook optional
6. Trigger first BuildRun
7. View BuildRun logs
8. Next steps: image build, GitOps release

Features:
- Detect empty state.
- Show setup progress.
- Provide copyable kubectl/API commands.
- Show errors clearly.
- Allow skipping advanced steps.
- Link to docs.

Tests:
- wizard renders on empty state
- project form validation
- repository form validation
- template selection
- trigger BuildRun
- error state

Acceptance criteria:
- npm run build passes.
- First-run wizard exists.
- Empty dashboard directs user to wizard.
- User can trigger first BuildRun through wizard.
- No hardcoded production API URL.
```

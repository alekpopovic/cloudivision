# Prompt 31 — Angular UX Upgrade for Real CI/CD Debugging

Phase: Frontend

```text
Continue working on the cloudivision project.

Task:
Upgrade the Angular + Tailwind UI from a basic dashboard to a useful CI/CD debugging interface.

Goal:
Make it easy for users to understand why a build or release failed.

Pages/components to improve:
1. BuildRun detail page:
   - step timeline
   - duration per step
   - condition timeline
   - failure summary
   - retry button
   - rerun with same params button
   - commit metadata section
   - related Release section
   - image digest section
   - Supply Chain tab if backend fields exist
2. Logs viewer:
   - auto-scroll toggle
   - pause refresh
   - search text
   - step filter if step metadata exists
   - copy selected logs
   - download logs if endpoint exists
   - ANSI color support if practical and safe
3. Repository onboarding wizard:
   - select provider, repo URL, default branch
   - select/create PipelineTemplate
   - show webhook URL and secret setup instructions
   - verify webhook status if endpoint exists
4. PipelineTemplate editor:
   - visual step list
   - add/remove/reorder steps
   - YAML editor mode if practical
   - validation errors
   - dry-run button if API supports it
   - build settings section
5. Release detail page:
   - GitOps commit link
   - PR link if PR-based promotion exists
   - Argo CD sync/health
   - approval history
   - deployment timeline
   - rollback placeholder/action if backend supports it
6. First-run wizard:
   - Create Project
   - Add Repository
   - Add PipelineTemplate
   - Trigger BuildRun
   - View logs
   - Optional configure GitOps

Implementation rules:
- Use Angular standalone components if project uses standalone style.
- Use Tailwind for layout.
- Keep API access in services.
- Use reactive forms.
- Use async pipe or takeUntilDestroyed.
- Keep components testable.
- Do not hide backend authorization errors.
- Do not hardcode localhost except documented dev default.

Tests:
- status badge tests
- BuildRun detail renders failure reason
- logs viewer handles loading/error/success
- repository wizard validates required fields
- release detail shows GitOps commit/PR links when present
- first-run wizard route renders

Acceptance criteria:
- npm run build passes.
- npm test passes if configured.
- BuildRun detail is meaningfully more useful for debugging.
- First-run wizard exists.
- Tailwind classes are used consistently.
- No large UI framework is introduced.
```

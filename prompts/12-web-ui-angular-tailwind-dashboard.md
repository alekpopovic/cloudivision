# Prompt 12 — Web UI: Angular + Tailwind Dashboard

Phase: Frontend

```text
Continue working on the cloudivision project.

Task:
Create the Angular + Tailwind CSS web UI for cloudivision in /web.

Stack:
- Angular
- TypeScript
- Angular Router
- Angular HttpClient
- RxJS
- Reactive Forms
- Tailwind CSS
- Minimal dependencies
- No heavy UI framework unless already chosen

Frontend structure:
- /web/src/app/core
- /web/src/app/shared
- /web/src/app/features/dashboard
- /web/src/app/features/projects
- /web/src/app/features/repositories
- /web/src/app/features/pipeline-templates
- /web/src/app/features/build-runs
- /web/src/app/features/environments
- /web/src/app/features/releases
- /web/src/app/api
- /web/src/environments

Pages:
1. Dashboard: project count, latest BuildRuns, failed builds, releases in progress.
2. Projects: list, create form, detail.
3. Repositories: list, create form, webhook URL.
4. Pipeline Templates: list, basic step editor, build settings form.
5. Build Runs: list, status badge, filters phase/project/repository, manual trigger form.
6. Build Run Detail: metadata, conditions timeline, image result, logs tab, auto-refresh logs.
7. Environments: list, Argo CD/Flux status if available.
8. Releases: list, approval placeholder, deployment status.

API client:
- Use Angular HttpClient through services.
- Support apiBaseUrl through Angular environment configuration and optionally /assets/config.json.
- Define TypeScript interfaces for Project, Repository, PipelineTemplate, BuildRun, Environment, Release and ApiError.
- Error handling shows backend code/message.

Angular rules:
- Use Angular Router.
- Prefer standalone components if generated that way.
- Use feature folders and reactive forms.
- Use async pipe or takeUntilDestroyed.
- Keep components simple and testable.

Tailwind requirements:
- App shell with sidebar, top bar, main content.
- Reusable components: StatusBadge, PageHeader, EmptyState, LoadingState, ErrorMessage, KeyValueList, ConditionsTimeline, LogsViewer.

UX:
- Support statuses Pending, Queued, Running, Succeeded, Failed, Cancelled, AwaitingApproval, Deploying, Deployed, RolledBack.
- Auto-refresh BuildRun list every 5 seconds and detail/logs every 2 seconds unless streaming exists.
- Clear empty states and backend errors.

Acceptance criteria:
- npm install works.
- npm run build passes.
- UI can run against local API server.
- BuildRun detail displays conditions and logs.
- Manual BuildRun form sends POST /api/v1/build-runs.
- No hardcoded localhost except documented dev default.
- Tailwind CSS is configured and used.
```

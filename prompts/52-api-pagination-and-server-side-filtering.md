# 52. API Pagination and Server-Side Filtering

```text
Continue working on the cloudivision project.

Task:
Add pagination and server-side filtering to API list endpoints.

Goal:
API and Angular UI should remain usable with thousands of BuildRuns and Releases.

Endpoints to update:
- GET /api/v1/build-runs
- GET /api/v1/releases
- GET /api/v1/audit/events
- GET /api/v1/projects if useful
- GET /api/v1/repositories if useful

Query params:
- limit
- continue/pageToken
- namespace
- project
- repository
- phase
- from
- to
- sort
- order

Response format:
{
  "items": [],
  "nextPageToken": "...",
  "totalCount": optional,
  "limit": 50
}

Requirements:
1. Default limit should be safe.
2. Maximum limit should be enforced.
3. Filtering should happen server-side where practical.
4. Use Kubernetes list options/labels/fields where practical.
5. Add indexes in controller-runtime cache where useful.
6. Angular UI must use pagination instead of rendering unbounded lists.
7. CLI should support --limit and --page-token where relevant.

Tests:
- default pagination
- max limit enforced
- phase filter
- project filter
- invalid query params
- Angular list uses paginated service call

Docs:
- docs/reference/api.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- BuildRun list is paginated.
- Angular UI does not request unlimited BuildRuns.
- API documentation updated.
```

# Prompt 37 — cloudivision CLI

Phase: CLI

```text
Continue working on the cloudivision project.

Task:
Create a cloudivision CLI.

Goal:
Provide a developer-friendly command-line interface for common operations.

Binary:
- /cmd/cloudivision

Use cobra if acceptable, otherwise standard flag package.

Commands:
- cloudivision version
- cloudivision login
- cloudivision project list/create
- cloudivision repo list/add
- cloudivision pipeline list
- cloudivision build trigger/list/get/logs/watch
- cloudivision release list/get/approve/reject/rollback
- cloudivision doctor

Configuration:
- API URL from --api-url, CLOU_DIVISION_API_URL or ~/.cloudivision/config.yaml.
- Auth token from --token, CLOU_DIVISION_TOKEN or config file.
- Namespace from --namespace, config file or default.

Command behavior:
1. build trigger accepts project, repository, pipelineTemplate, revision, branch and params; creates BuildRun through API; prints BuildRun name; optional --watch.
2. build logs streams or polls logs; supports --tail and --follow.
3. build watch polls BuildRun status and exits 0 on Succeeded, non-zero on Failed/Cancelled/timeout.
4. doctor checks API, auth, CRDs if Kubernetes access exists, controller/API/web deployments, runner image, RBAC, provider health and GitOps provider health.

Output:
- human-readable table by default.
- --output json support for key commands.

Tests:
- CLI config loading
- build trigger request generation
- build watch success/failure
- doctor output with fake API

Docs:
- docs/reference/cli.md
- README quickstart mentions CLI

Acceptance criteria:
- go test ./... passes.
- cloudivision version works.
- cloudivision build trigger can create a BuildRun through API.
- cloudivision build watch exits correctly.
- cloudivision doctor gives actionable output.
- docs/reference/cli.md exists.
```

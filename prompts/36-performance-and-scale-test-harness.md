# Prompt 36 — Performance and Scale Test Harness

Phase: Scale

```text
Continue working on the cloudivision project.

Task:
Create a performance and scale test harness for cloudivision.

Goal:
Understand how cloudivision behaves under higher BuildRun, webhook and release load.

Create:
- /test/scale/run.sh
- /test/scale/generate-buildruns.go or equivalent
- /test/scale/README.md
- Makefile target: scale-test

Scenarios:
1. 100 BuildRuns in one namespace
2. 1000 BuildRuns across 20 projects
3. 100 webhook events in 10 seconds
4. 50 concurrent runner Jobs
5. 10 concurrent GitOps Releases
6. Large logs: 100MB per BuildRun if practical
7. UI/API list with many BuildRuns

Measurements:
- controller memory and CPU
- API latency and memory
- Kubernetes API request rate if observable
- reconcile queue depth if metric exists
- BuildRun completion duration
- log endpoint latency
- UI list rendering performance
- PostgreSQL audit write latency if enabled

Required improvements if missing:
1. API pagination for BuildRun list with safe default page size.
2. Server-side filtering by phase, project, repository, namespace and time range if supported.
3. Controller indexes for common lookups: BuildRun by project/repository, Release by BuildRun, resources by eventID if feasible.
4. Configurable controller concurrency and per-project/global runner concurrency design or implementation.

Docs:
- test/scale/README.md explains prerequisites, scenarios, interpreting results, safe cluster sizing and known bottlenecks.

Acceptance criteria:
- make scale-test exists.
- Scale tests can generate BuildRuns safely with cleanup.
- API supports pagination for BuildRun list.
- Angular UI does not render thousands of rows at once.
- Results template exists or first run is documented.
- go test ./... passes.
- npm run build passes in /web.
```

# Prompt 34 — Upgrade, Backup, Restore and Uninstall

Phase: Operations

```text
Continue working on the cloudivision project.

Task:
Add operational lifecycle documentation and tests for upgrade, backup, restore and uninstall.

Goal:
Make cloudivision safe to operate beyond a local demo.

Docs to add:
- docs/operations/upgrade.md
- docs/operations/backup-restore.md
- docs/operations/uninstall.md
- docs/operations/disaster-recovery.md

Upgrade docs:
- upgrading Helm chart
- upgrading CRDs
- preserving custom resources
- checking controller/API health
- rollback plan
- known limitations for v1alpha1

Backup docs:
- backing up CRDs and CRs
- backing up PostgreSQL audit DB if enabled
- GitOps repo is external responsibility
- backing up secrets or using external secret manager
- restoring into a new cluster

Uninstall docs:
- uninstalling Helm release
- preserving CRDs by default
- optional CRD deletion
- consequences of deleting CRDs
- cleanup of namespaces created by Project controller

Disaster recovery docs:
- controller/API lost
- runner Job stuck
- GitOps repo unavailable
- registry unavailable
- PostgreSQL unavailable
- cluster restore

Test scripts:
- /test/upgrade/run.sh
- /test/upgrade/fixtures/
- Makefile target: upgrade-test

Upgrade test skeleton:
1. Install current chart.
2. Create Project, Repository, PipelineTemplate, BuildRun.
3. Run a successful BuildRun.
4. Simulate upgrade by applying chart again or changing image tag.
5. Assert existing resources still exist.
6. Assert controller still reconciles new BuildRun.
7. Assert uninstall does not delete CRDs unless explicitly requested.

Acceptance criteria:
- Operations docs exist.
- make upgrade-test exists, even if some parts are initially skipped with explicit messages.
- Helm uninstall behavior around CRDs is clearly documented.
- Existing resources survive chart upgrade in test or documented manual procedure.
- go test ./... passes.
```

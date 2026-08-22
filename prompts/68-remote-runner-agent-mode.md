# 68. Remote Runner / Agent Mode

```text
Continue working on the cloudivision project.

Task:
Design remote runner/agent mode.

Goal:
Prepare cloudivision for scenarios where runners execute outside the control-plane cluster or in remote clusters.

Create:
- docs/architecture/remote-runner.md
- docs/security/remote-runner.md

Possible components:
- runner agent
- API polling mode
- runner registration
- short-lived token
- runner groups
- runner labels
- project-to-runner matching

CRD/API concepts:
RunnerPool:
- name
- type: kubernetes-job, remote-agent
- labels
- maxConcurrentRuns
- status

Security model:
- remote agent must not receive global secrets
- token rotation
- per-project runner authorization
- network restrictions
- audit all assignments

Implementation:
- Add interfaces only if practical.
- Do not break current Kubernetes Job executor.
- Add skeleton provider for remote runner marked experimental.

Acceptance criteria:
- Remote runner architecture docs exist.
- Security risks are documented.
- Current Job executor remains default.
- Any code added is clearly experimental and tested.
```

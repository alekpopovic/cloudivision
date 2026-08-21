# Prompt 21 — v0.1 Product Readiness Audit

Phase: v0.1 Hardening

```text
Continue working on the cloudivision project.

Context:
The initial 20 implementation prompts have already been completed. The project now needs to move from “implemented prototype” to a reliable v0.1 platform.

Task:
Perform a full v0.1 product readiness audit.

Goal:
Determine whether a new user can clone the repository, install cloudivision into a clean Kubernetes cluster, create the first BuildRun, view logs through the API/UI and understand failures without reading source code.

Audit areas:
1. Fresh clone readiness
- Can a developer clone the repository and run the documented commands?
- Are prerequisites documented?
- Are generated files committed where needed?
- Are scripts executable?
- Does make help or equivalent explain available commands?

2. Local kind install
- Does hack/kind-create.sh work on a clean machine with kind installed?
- Does hack/install-dev.sh install CRDs, controller, API and web UI?
- Are local images built and loaded correctly?
- Are namespaces created consistently?

3. Helm install
- Does helm template charts/cloudivision work?
- Does helm install work on a clean cluster?
- Are values documented?
- Are default values safe?

4. CRD health
- Are CRDs generated from current Go types?
- Are CRD schemas valid?
- Are sample CRs accepted by Kubernetes?
- Do status subresources work?

5. Runtime flow
- Can a Project, Repository, PipelineTemplate and BuildRun be created?
- Does BuildRun create a runner Job?
- Does runner update status?
- Can logs be fetched through API?
- Does Angular UI show the run?

6. Security baseline
- No privileged containers by default.
- No docker.sock mount.
- No hostPath mount unless explicitly justified.
- Runner ServiceAccount is least-privilege.
- Webhook secrets are verified.
- Secrets are redacted in logs.

7. Documentation
- README quickstart is accurate.
- First BuildRun guide exists.
- First webhook guide exists.
- First GitOps release guide exists.
- Troubleshooting page exists.

Deliverables:
1. Create docs/readiness/v0.1-readiness-audit.md.
2. Add a checklist with pass/fail/unknown status for every item.
3. Open TODO comments or GitHub issue references for failed/unknown items if issue tracking exists.
4. Fix critical documentation or script problems found during the audit if they are safe and small.
5. Do not add new product features in this task.

Acceptance criteria:
- docs/readiness/v0.1-readiness-audit.md exists.
- The audit clearly separates passed, failed, unknown checks and recommended fixes.
- Critical blockers for v0.1 are listed at the top.
- Safe quick fixes are implemented.
- At the end, summarize changed files, tests run and remaining risks.
```

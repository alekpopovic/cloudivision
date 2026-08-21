# Prompt 38 — Documentation Restructure for Users and Operators

Phase: Documentation

```text
Continue working on the cloudivision project.

Task:
Restructure documentation so it is useful for new users, operators, security reviewers and contributors.

Goal:
Move from implementation notes to user-facing documentation.

Create or reorganize docs into:

docs/getting-started/
- quickstart-kind.md
- first-build.md
- first-webhook.md
- first-release.md

docs/concepts/
- project.md
- repository.md
- pipeline-template.md
- buildrun.md
- release.md
- environment.md
- provider.md
- policy.md

docs/operations/
- install-helm.md
- upgrade.md
- backup-restore.md
- observability.md
- troubleshooting.md
- security-hardening.md

docs/security/
- threat-model.md
- runner-security.md
- supply-chain.md
- auth-rbac.md
- webhook-security.md

docs/development/
- architecture.md
- controllers.md
- runner.md
- api.md
- frontend.md
- testing.md
- contributing.md
- adding-provider.md

docs/reference/
- api.md
- crds.md
- helm-values.md
- cli.md
- configuration.md

Requirements:
1. Every getting-started guide includes exact commands.
2. Every concept page includes a small YAML example.
3. Every operation page includes troubleshooting guidance.
4. Security docs clearly state current limitations.
5. Development docs explain how to run tests.
6. Frontend docs explain Angular + Tailwind local development.
7. API reference describes JSON error format.
8. Helm values reference is generated or kept in sync with values.yaml.
9. Add docs/index.md as landing page.

Examples to include:
- Node.js app
- Go app
- Java/Maven app if practical
- Python app if practical
- Dockerfile build
- Helm values GitOps release
- Kustomize GitOps release
- production approval
- failed build troubleshooting

Acceptance criteria:
- docs/index.md exists.
- Documentation follows the proposed structure.
- README links to docs/index.md and quickstart.
- Old docs are either moved or linked.
- No important feature is documented only in code comments.
- Broken or outdated commands are fixed.
```

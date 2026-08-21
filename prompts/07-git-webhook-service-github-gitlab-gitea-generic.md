# Prompt 07 — Git Webhook Service: GitHub, GitLab, Gitea, Generic

Phase: API

```text
Continue working on the cloudivision project.

Task:
Add webhook receivers to the API server for Git events.

Endpoints:
- POST /api/v1/webhooks/github/{repositoryName}
- POST /api/v1/webhooks/gitlab/{repositoryName}
- POST /api/v1/webhooks/gitea/{repositoryName}
- POST /api/v1/webhooks/generic/{repositoryName}

Behavior:
1. Find Repository CR by repositoryName.
2. Verify webhook secret:
   - GitHub: X-Hub-Signature-256 HMAC SHA256
   - GitLab: X-Gitlab-Token
   - Gitea: X-Gitea-Signature if supported, otherwise documented token mode
   - Generic: Authorization: Bearer or X-Cloudivision-Token
3. Parse repository URL, branch, commit SHA, actor and event ID.
4. Check branch matches Repository.spec.defaultBranch or supported filter.
5. On push event, create BuildRun with projectRef, repositoryRef, pipelineTemplateRef, revision/commitSHA and triggeredBy webhook metadata.
6. Idempotency: if same eventID already has a BuildRun, do not create duplicate; return existing BuildRun.
7. Add audit event through internal/audit placeholder.
8. Add test fixtures for GitHub and GitLab payloads.

Security:
- Never accept GitHub/GitLab webhook without verification when Repository.spec.webhook.enabled=true.
- Do not log full payload.
- Limit request body size.
- Return 401/403 for invalid signature.

Acceptance criteria:
- go test ./... passes.
- Signature verification has unit tests.
- Valid push payload creates a BuildRun.
- Repeated event does not create duplicate BuildRun.
- Invalid signature creates nothing.
- Body size limit exists.
```

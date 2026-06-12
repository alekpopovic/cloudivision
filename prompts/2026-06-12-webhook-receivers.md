# Webhook receivers

User prompt:

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
1. Find the Repository CR by repositoryName.
2. Verify webhook secret:
   - GitHub: X-Hub-Signature-256 HMAC SHA256
   - GitLab: X-Gitlab-Token
   - Gitea: X-Gitea-Signature if supported by implementation, otherwise documented token mode
   - Generic: Authorization: Bearer or X-Cloudivision-Token
3. Parse payload:
   - repository URL
   - branch
   - commit SHA
   - actor
   - event ID
4. Check whether branch matches Repository.spec.defaultBranch or a supported filter.
5. On push event, create BuildRun:
   - projectRef from Repository
   - repositoryRef = Repository name
   - pipelineTemplateRef from Repository spec
   - revision/commitSHA from payload
   - triggeredBy.type = webhook
   - triggeredBy.actor
   - triggeredBy.eventID
6. Idempotency:
   - if the same eventID already has a BuildRun, do not create a duplicate
   - return the existing BuildRun in the response
7. Add audit event through internal/audit placeholder.
8. Add test fixtures for GitHub and GitLab payloads.

Security:
- Never accept GitHub/GitLab webhook without verification when Repository.spec.webhook.enabled=true.
- Do not log the full payload because it may contain tokens or private metadata.
- Limit request body size.
- Return 401/403 for invalid signature.

Acceptance criteria:
- go test ./... passes.
- Signature verification has unit tests.
- Valid push payload creates a BuildRun.
- Repeated event does not create a duplicate BuildRun.
- Invalid signature creates nothing.
- Body size limit exists.
```

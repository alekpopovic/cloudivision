# 46. GitHub Webhook Production Readiness

```text
Continue working on the cloudivision project.

Task:
Make GitHub webhook handling production-ready.

Improve endpoint:
- POST /api/v1/webhooks/github/{repositoryName}

Requirements:
1. Verify X-Hub-Signature-256 using HMAC SHA256.
2. Reject missing signatures when webhook is enabled.
3. Reject invalid signatures.
4. Enforce body size limit.
5. Parse:
   - push events
   - pull_request events
   - ping events
6. Support GitHub delivery ID idempotency.
7. Store processed delivery IDs through audit/webhook idempotency backend.
8. Return stable JSON responses.
9. Do not log full payload.
10. Add replay protection if timestamp/event data allows it.
11. Add useful audit events:
   - WebhookAccepted
   - WebhookRejected
   - WebhookDuplicate
   - BuildRunCreatedFromWebhook

Tests:
- valid push
- invalid signature
- missing signature
- duplicate delivery ID
- ignored branch
- ping event
- malformed payload
- body too large

Docs:
- docs/getting-started/first-webhook.md
- docs/security/webhook-security.md

Acceptance criteria:
- go test ./... passes.
- Invalid signature never creates BuildRun.
- Duplicate webhook does not create duplicate BuildRun.
- GitHub ping returns success without creating BuildRun.
- Docs include exact GitHub webhook setup steps.
```

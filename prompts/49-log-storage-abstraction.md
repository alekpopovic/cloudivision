# 49. Log Storage Abstraction

```text
Continue working on the cloudivision project.

Task:
Add a log storage abstraction.

Goal:
Kubernetes pod logs are not enough for long-term CI/CD history. Add a pluggable log backend.

Backends:
- kubernetes-pod-logs default
- local filesystem for dev
- object storage skeleton
- Loki skeleton

Create:
- /internal/logstore

Interface:
type LogStore interface {
    Append(ctx context.Context, req AppendLogRequest) error
    Read(ctx context.Context, req ReadLogRequest) (*ReadLogResult, error)
    Stream(ctx context.Context, req StreamLogRequest) (<-chan LogLine, error)
    Delete(ctx context.Context, req DeleteLogRequest) error
}

Behavior:
1. Runner writes logs to configured log backend if enabled.
2. API reads logs from log backend first.
3. Fallback to Kubernetes pod logs if no backend configured.
4. Logs should support:
   - tail
   - follow if practical
   - step filter if metadata exists
   - timestamps
5. Apply secret redaction before persisting logs.

CRD/status:
- BuildRun.status.log.backend
- BuildRun.status.log.ref

Angular UI:
- Logs viewer should work with stored logs.
- Add search and pause refresh if not already present.

Docs:
- docs/operations/logs.md

Tests:
- memory/local log store
- redaction before append
- tail behavior
- pod log fallback

Acceptance criteria:
- go test ./... passes.
- Logs can be read after runner pod finishes if backend is enabled.
- Default behavior still works without external log store.
- Secrets are redacted before storage.
```

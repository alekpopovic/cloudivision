# 47. Branch, Tag and Pull Request Event Filters

```text
Continue working on the cloudivision project.

Task:
Add event filtering for branches, tags and pull requests.

Goal:
Users should control which Git events trigger BuildRuns.

Extend Repository.spec.webhook:
- branchFilters:
  - include []string
  - exclude []string
- tagFilters:
  - include []string
  - exclude []string
- pullRequest:
  - enabled bool
  - events []string
  - buildForks bool
  - requireTrustedActor bool

Behavior:
1. Push to included branch creates BuildRun.
2. Push to excluded branch is ignored.
3. Tag push can trigger BuildRun if enabled.
4. Pull request opened/synchronize can trigger BuildRun if enabled.
5. Forked PRs are blocked by default unless buildForks=true.
6. Trusted actor logic should be skeleton or implemented if auth model supports it.
7. Response should explain ignored events.

Tests:
- include branch match
- exclude branch match
- tag trigger
- PR trigger
- forked PR blocked
- unknown event ignored

Angular UI:
- Repository form should allow basic branch include/exclude filters.
- Show webhook event filter summary on Repository detail.

Docs:
- docs/concepts/repository.md
- docs/getting-started/first-webhook.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Filters are documented.
- Ignored events are auditable.
```

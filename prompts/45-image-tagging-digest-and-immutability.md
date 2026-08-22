# 45. Image Tagging, Digest and Immutability

```text
Continue working on the cloudivision project.

Task:
Implement a robust image tagging and digest strategy.

Goal:
Avoid unsafe mutable image deployment patterns.

Image tag strategy:
- commit SHA tag
- branch-shortSHA tag
- BuildRun name tag
- semver tag when manually provided
- latest disabled for production by default

Add config:
- default tag template
- allow latest bool
- require digest for Release bool

Suggested tag templates:
- "{{ .Branch }}-{{ .ShortSHA }}"
- "{{ .BuildRunName }}"
- "{{ .CommitSHA }}"
- "{{ .Timestamp }}-{{ .ShortSHA }}"

Behavior:
1. If BuildRun.spec.image.tag is empty, generate tag.
2. Sanitize tag to registry-compatible format.
3. Store generated tag in BuildRun.status.image.tag.
4. Always prefer digest for Release.
5. Block production Release if digest is missing and policy requires it.
6. UI should show tag and digest separately.

Tests:
- tag generation
- branch sanitization
- empty commit handling
- latest blocked for production
- digest required policy

Docs:
- docs/build/image-tagging.md

Acceptance criteria:
- go test ./... passes.
- Image tags are deterministic and safe.
- Production Release can require digest.
- Angular UI shows digest clearly.
- Docs explain recommended tagging strategy.
```

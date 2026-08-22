# 63. CLI Packaging and Distribution

```text
Continue working on the cloudivision project.

Task:
Prepare cloudivision CLI for distribution.

Goal:
Make the CLI installable and versioned.

Improve:
- cmd/cloudivision
- version command
- build metadata injection
- shell completions
- install script
- release artifacts

Add:
- goreleaser config if acceptable
- Makefile target:
  - build-cli
  - install-cli-local
  - cli-completions

CLI install docs:
- macOS
- Linux
- Windows via binary download
- container usage if useful

Commands to verify:
- cloudivision version
- cloudivision doctor
- cloudivision build trigger
- cloudivision build logs
- cloudivision build watch
- cloudivision release approve

Tests:
- config loading
- API URL resolution
- token resolution
- command error formatting
- JSON output

Acceptance criteria:
- go test ./... passes.
- CLI binary builds.
- Version command includes version, commit, date.
- Docs explain installation.
- Release process includes CLI artifact.
```

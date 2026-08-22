# 73. More Real-World Examples

```text
Continue working on the cloudivision project.

Task:
Add real-world example applications and pipeline templates.

Goal:
Make cloudivision easy to evaluate with realistic projects.

Examples:
- examples/go-api
- examples/node-api
- examples/angular-app
- examples/python-fastapi
- examples/java-maven
- examples/dockerfile-only
- examples/helm-release
- examples/kustomize-release

Each example should include:
- app source or minimal fixture
- Dockerfile if applicable
- tests
- PipelineTemplate
- Repository example
- BuildRun example
- optional Release example
- README

Docs:
- docs/examples/index.md
- link examples from getting started

Conformance:
- Use at least one example in conformance tests.
- Use Angular example for dogfood-style frontend build if useful.

Acceptance criteria:
- Examples exist.
- At least Go, Node.js, Angular and Dockerfile examples work.
- Example docs include exact commands.
- Examples do not require paid external services by default.
```

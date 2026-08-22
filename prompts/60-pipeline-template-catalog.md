# 60. Pipeline Template Catalog

```text
Continue working on the cloudivision project.

Task:
Add a built-in PipelineTemplate catalog.

Goal:
Make it easy for users to start with common stacks.

Catalog templates:
- Go
- Node.js npm
- Node.js pnpm
- Angular
- React
- Python
- Java Maven
- Java Gradle
- Dockerfile-only
- Helm chart validation
- Kubernetes manifest validation

Create:
- /deploy/catalog/
- /docs/catalog/
- API endpoint:
  - GET /api/v1/catalog/pipeline-templates
  - POST /api/v1/catalog/pipeline-templates/{name}/install

Behavior:
1. Catalog templates are versioned.
2. User can install template into namespace/project.
3. Template parameters are documented.
4. Angular UI shows template catalog.
5. User can create PipelineTemplate from catalog.

Tests:
- catalog loads
- install template
- invalid template name
- Angular catalog page renders

Docs:
- docs/catalog/index.md
- one doc per template

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- At least Go, Node.js, Angular and Dockerfile templates exist.
- UI can create a PipelineTemplate from catalog.
```

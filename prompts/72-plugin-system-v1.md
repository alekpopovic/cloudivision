# 72. Plugin System v1

```text
Continue working on the cloudivision project.

Task:
Design and implement the first version of the cloudivision plugin system.

Goal:
Allow integrations to be added without hardcoding everything into the core.

Plugin types:
- git provider
- registry provider
- build provider
- gitops provider
- notification provider
- policy provider
- supply-chain provider

Start simple:
- In-process plugin registry.
- Static plugin registration at startup.
- External runtime plugins are future work.

Plugin metadata:
- name
- type
- version
- capabilities
- config schema
- health check
- docs URL optional

API:
- GET /api/v1/plugins
- GET /api/v1/plugins/{type}/{name}
- GET /api/v1/plugins/health

Angular UI:
- Plugins page
- health and capability display
- configuration status

Docs:
- docs/plugins/index.md
- docs/plugins/creating-plugin.md

Tests:
- plugin registration
- duplicate plugin
- health check
- unsupported plugin
- API list

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Provider registry is compatible with plugin model.
- Plugins are visible through API/UI.
- External plugins are explicitly documented as future work.
```

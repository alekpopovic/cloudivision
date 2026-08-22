# API development

The API is implemented with Go's HTTP stack in `internal/api`. Handlers translate
between stable JSON DTOs and Kubernetes resources, apply authorization and policy,
then record audit events. Avoid returning CRD structs directly.

Run the API and tests:

```sh
go test ./internal/api/...
KUBECONFIG="$KUBECONFIG" CLOU_DIVISION_AUTH_MODE=disabled go run ./cmd/api
curl -fsS http://localhost:8080/healthz
```

`disabled` authentication is development-only. For production configure OIDC,
group mappings, CORS and TLS at the ingress. Every error response must remain JSON:

```json
{"code":"bad_request","message":"projectRef is required","requestId":"req-123"}
```

Policy errors may also include `violations`. When changing a route, update handler
tests, `docs/openapi.yaml`, [API reference](../reference/api.md), the CLI and UI
client where applicable.

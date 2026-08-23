# API client generation

`docs/openapi.yaml` is the source of truth for public HTTP routes, shared error
responses and pagination. It covers authentication, BuildRun and Release
actions, providers, organizations, audit export and compliance reports.

Regenerate committed clients after changing a route:

```bash
make sdk-generate
make sdk-check
```

The generator in `cmd/sdkgen` validates required schemas and checks every
`Server.Handler` route is represented in OpenAPI. It emits:

- `sdk/go/client.gen.go`: a context-aware HTTP client used by CLI/external Go
  integrations, including bearer token and organization headers;
- `sdk/typescript/client.gen.ts`: typed operation names and a fetch-based client
  suitable for Angular or other TypeScript consumers.

`make sdk-check` fails if committed output is stale, compiles the Go SDK and
type-checks the TypeScript SDK with the web workspace compiler. CI runs it.

The Angular application still uses its existing injectable `ApiClient`, and the
CLI still uses its current internal transport. Migrate endpoint-by-endpoint to
the generated clients; keep UI-specific RxJS mapping and CLI presentation out
of generated code. Do not manually edit `*.gen.go` or `*.gen.ts`.

All non-2xx responses use `{code, message, requestId?, violations?}`. Generated
clients expose the stable code/message/request ID fields; callers must tolerate
additional fields. Paginated list responses use `{items, nextPageToken?,
totalCount, limit}` and continuation tokens are opaque.

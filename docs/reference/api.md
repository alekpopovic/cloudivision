# HTTP API reference

The API is served below `/api/v1`. The machine-readable contract is
[`docs/openapi.yaml`](../openapi.yaml). Health and readiness are available at
`/healthz` and `/readyz`.

## Resources

| Route | Methods | Purpose |
| --- | --- | --- |
| `/api/v1/projects` | `GET`, `POST` | List and create projects |
| `/api/v1/projects/{name}` | `GET` | Read a project |
| `/api/v1/repositories` | `GET`, `POST` | List and create repositories |
| `/api/v1/pipeline-templates` | `GET`, `POST` | List and create templates |
| `/api/v1/build-runs` | `GET`, `POST` | List and trigger builds |
| `/api/v1/build-runs/{namespace}/{name}` | `GET` | Read a build |
| `/api/v1/build-runs/{namespace}/{name}/logs` | `GET` | Stream runner logs |
| `/api/v1/environments` | `GET` | List environments |
| `/api/v1/releases` | `GET` | List releases |
| `/api/v1/releases/{namespace}/{name}/approve` | `POST` | Approve a release |
| `/api/v1/releases/{namespace}/{name}/reject` | `POST` | Reject a release |
| `/api/v1/audit/events` | `GET` | List audit events |
| `/api/v1/providers` | `GET` | List provider capabilities |
| `/api/v1/providers/health` | `GET` | Read provider health |
| `/api/v1/webhooks/{provider}/{repository}` | `POST` | Receive a signed webhook |

## Pagination and filtering

BuildRun, Release, and audit-event lists return a page envelope:

```json
{
  "items": [],
  "nextPageToken": "eyJ2IjoxLCJvIjo1MH0",
  "totalCount": 240,
  "limit": 50
}
```

The default limit is 50 and the maximum is 200. Pass the opaque
`nextPageToken` value back as `pageToken` (or `continue`) to request the next
page. Tokens must not be parsed by clients. The older numeric `offset` query is
accepted temporarily for v0.1 client compatibility, but new clients should use
tokens.

Common parameters are `limit`, `pageToken`, `namespace`, `project`,
`repository`, `phase`, `from`, `to`, `sort`, and `order`. Times use RFC3339 and
are inclusive. `order` is `asc` or `desc`. BuildRuns and Releases support
`createdAt`, `name`, and `phase` sorting; audit events support `createdAt` and
`type`. BuildRuns also accept the deprecated `createdAfter` and
`createdBefore` aliases. Filters that do not apply to a resource are ignored.

Examples:

```sh
curl -fsS 'http://localhost:8080/api/v1/build-runs?namespace=ci&project=payments&phase=Failed&limit=50'
curl -fsS 'http://localhost:8080/api/v1/releases?project=payments&sort=createdAt&order=desc&limit=25'
curl -fsS 'http://localhost:8080/api/v1/audit/events?repository=checkout&from=2026-08-01T00:00:00Z&limit=100'
```

## Authentication and errors

Send an OIDC bearer token when auth is enabled:

```sh
curl -fsS http://localhost:8080/api/v1/build-runs \
  -H "Authorization: Bearer $CLOU_DIVISION_TOKEN"
```

All API failures use JSON, including authentication and authorization failures:

```json
{
  "code": "policy_denied",
  "message": "build is denied by policy",
  "requestId": "9f43b",
  "violations": [
    {"code": "privileged_build_denied", "message": "privileged builds are disabled", "field": "spec.build.privileged"}
  ]
}
```

`code` is stable enough for clients; `message` is human readable; `requestId`
correlates logs. `violations` is optional and reports policy fields. Never parse the
message to decide program behavior.

Production approval uses the authenticated principal when available:

```sh
curl -fsS -X POST http://localhost:8080/api/v1/releases/team-a/release-1/approve \
  -H 'Content-Type: application/json' \
  -d '{"actor":"operator@example.com","comment":"change reviewed"}'
```

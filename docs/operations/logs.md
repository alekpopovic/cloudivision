# Build logs

The default backend is `kubernetes-pod-logs`. It requires no external service,
but logs disappear when Kubernetes removes the runner Pod. The API keeps the
existing `tailLines` behavior and reads Pod logs when no persistent backend is
configured.

For local development, set:

```sh
export CLOU_DIVISION_LOG_BACKEND=local
export CLOU_DIVISION_LOG_ROOT="$PWD/.data/logs"
```

The runner appends JSON Lines files with mode `0600`; the API reads the same
directory before attempting any Pod lookup. In Kubernetes, configure a shared
PVC so finished Job logs remain readable:

```yaml
logs:
  backend: local
  local:
    root: /var/lib/cloudivision/logs
    existingClaim: cloudivision-logs
```

`BuildRun.status.log.backend` and `BuildRun.status.log.ref` record where stored
logs live. `GET /api/v1/build-runs/{namespace}/{name}/logs` supports
`tailLines` and an optional `step` query. Responses contain timestamped lines;
the Angular viewer supports search, step filtering, pause, copy, and download.

Runner output is redacted before `Append`: known secret environment values,
credential-bearing repository URLs, and secret-like key/value pairs are masked.
This is defense in depth, not permission to print secrets. Repository code is
untrusted and must never deliberately echo credentials.

The `object` and `loki` backend types are interface-compatible skeletons and
return a clear not-implemented error. Do not select them in production until a
credential-scoped adapter, retention policy, encryption, and integration tests
are provided.

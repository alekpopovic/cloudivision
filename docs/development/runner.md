# Runner development

The runner clones untrusted repository code, executes declared steps, optionally
builds an OCI image through rootless BuildKit, and emits status for the controller.
It must not mount the Docker socket, run privileged by default, expose credentials,
or apply workloads to destination namespaces.

Run its focused and command-package tests locally:

```sh
go test ./internal/runner/... ./internal/build/...
go test ./cmd/runner
```

The Kubernetes Job executor supplies runner configuration through environment
variables and mounted Secrets. Treat additions as a security boundary: validate
paths and inputs, preserve command timeouts, redact output, and add resource
limits. See [runner security](../security/runner-security.md).

Image builds require `buildctl` or `buildctl-daemonless.sh` and an intentionally
configured rootless BuildKit service. A developer pipeline can set
`spec.build.enabled: false` when testing clone and step execution only.

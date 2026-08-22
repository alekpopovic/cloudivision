# Rootless BuildKit image builds

cloudivision uses BuildKit through the image-builder interface. Image building is
optional: a PipelineTemplate with `spec.build.enabled: false` never requires
BuildKit. The supported path does not mount `docker.sock`, does not use
Docker-in-Docker, and does not require a privileged runner.

## Runtime prerequisites

The runner image must contain either `buildctl-daemonless.sh` or `buildctl` and
must be able to reach an intentionally configured rootless BuildKit daemon. The
adapter prefers the daemonless script when both executables are present. Rootless
BuildKit still depends on kernel, snapshotter, storage, and Kubernetes Pod
Security configuration; verify the selected rootless mode on the target cluster.

Keep the runner non-root, set resource requests/limits and a pipeline timeout,
and grant it only the BuildRun status and named credential access it needs. Never
solve a BuildKit setup problem by mounting the Docker socket or enabling a
privileged default.

## Pipeline configuration

```yaml
spec:
  build:
    enabled: true
    builder: buildkit
    contextDir: services/api
    dockerfile: docker/release.Dockerfile
    image: registry.example.com/team/api
    push: true
    buildArgs:
      GO_VERSION: "1.26"
    target: release
    platforms: [linux/amd64, linux/arm64]
    labels:
      org.opencontainers.image.source: https://github.com/example/api
    cache:
      enabled: true
      mode: registry
      ref: registry.example.com/team/api:buildcache
```

`contextDir` is resolved under the cloned repository. It must exist and be a
directory. `dockerfile` is relative to that context, must exist, and cannot
escape the context with an absolute path or `..`. Build arguments and labels are
passed as Dockerfile frontend options in stable key order. `target` selects a
Dockerfile stage. `platforms` is a set of OCI platform strings passed as one
multi-platform request.

Cache is disabled by default:

- `inline` exports inline cache metadata and needs no `ref`;
- `registry` imports/exports the configured registry cache reference; and
- `local` imports/exports a configured runner-local path and should be used only
  with an intentionally isolated persistent workspace.

Registry and local cache modes require `cache.ref`. Cache content is untrusted
build input; do not share writable caches across mutually untrusted projects.

## Push and digest behavior

BuildKit receives an image exporter with the requested repository, tag, and
`push` flag. It also receives a private metadata-file path. After a successful
push, the adapter requires a valid `sha256` manifest or index digest from that
metadata and stores repository, tag, and digest separately in
`BuildRun.status.image`.

With `push: false`, a build may succeed without a digest because no registry
manifest was created. The status condition `ImagePushed=False` uses reason
`PushDisabled`, and `ImageDigestCaptured=False` uses `ImageDigestUnavailable`.
With `push: true`, missing or malformed digest metadata fails the BuildRun with
`ImageDigestCaptureFailed` rather than reporting an ambiguous successful push.

The runner records these stage conditions:

- `ImageBuildStarted`
- `ImageBuilt`
- `ImagePushed`
- `ImageDigestCaptured`

Downstream production Releases should use `repository@digest`, not a mutable tag.

## Failure reasons

| Reason | Meaning |
|---|---|
| `BuildKitUnavailable` | Neither supported BuildKit executable can be run. |
| `ImageBuildInputInvalid` | Repository, path, or cache configuration is invalid. |
| `ImageBuildContextMissing` | The configured context directory does not exist. |
| `ImageBuildDockerfileMissing` | The Dockerfile does not exist inside the context. |
| `ImageBuildFailed` | BuildKit failed, was cancelled, or exceeded its timeout. |
| `ImageDigestCaptureFailed` | A pushed build did not yield valid digest metadata. |

BuildKit diagnostics are bounded before they enter the BuildRun failure message.
Do not pass credentials as build arguments or labels: those are build inputs and
may be preserved in image metadata or tool output. Registry credentials belong
in scoped Kubernetes Secrets and are covered by the registry-provider workflow.

## Local verification

Run the adapter and runner tests without needing a live daemon:

```sh
go test ./internal/build/... ./internal/runner/...
```

An end-to-end push additionally requires a disposable authenticated registry and
a rootless BuildKit service. Verify the recorded digest exists remotely and run
the rendered workload security checks after changing runner images or Helm
values.

# Runner security

Treat every repository, branch, Dockerfile, dependency script, and build output as hostile. Default Jobs run non-root, drop capabilities, disable privilege escalation, use RuntimeDefault seccomp, avoid hostPath and Docker socket, and receive resource limits/deadlines.

```yaml
spec:
  resources: {cpuRequest: 100m, cpuLimit: 1, memoryRequest: 128Mi, memoryLimit: 1Gi, timeoutSeconds: 900}
  security: {allowPrivileged: false, runAsNonRoot: true, readOnlyRootFilesystem: false}
```

The policy engine denies privileged runners and Docker socket/hostPath references. `CLOU_DIVISION_ALLOW_PRIVILEGED_BUILDS` is an emergency compatibility switch, not a production recommendation. Prefer rootless BuildKit and scoped registry/Git credentials. The runner can get/update only its BuildRun and must not enumerate Secrets or deploy application manifests.

Current limitation: generated step execution shares a runner process/workspace, and default network egress may be broader than a hostile multi-tenant system requires. Use isolated nodes/runtime classes, namespace quota, egress allowlists, short-lived credentials, admission policy, and log/artifact retention controls for stronger boundaries.

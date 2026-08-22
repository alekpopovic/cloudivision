# PipelineTemplate

A PipelineTemplate defines ordered steps, optional image build, resources, timeout, runner security, params, and supply-chain hooks. It stays independent from a particular commit.

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: PipelineTemplate
metadata: {name: go-ci, namespace: ci}
spec:
  projectRef: platform
  steps:
    - {name: test, image: golang:1.26, command: [go], args: [test, ./...], timeoutSeconds: 300}
  build: {enabled: false, builder: none, push: false}
  resources: {cpuRequest: 100m, cpuLimit: "1", memoryRequest: 128Mi, memoryLimit: 1Gi, timeoutSeconds: 600}
  security: {allowPrivileged: false, runAsNonRoot: true, readOnlyRootFilesystem: false}
```

The default policy denies privileged, Docker socket, and hostPath requests. Images should be pinned; mutable step tags are currently allowed but reduce reproducibility.

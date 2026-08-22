# First build

After the kind quickstart, create the standard Node.js example and trigger it:

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f deploy/examples/repository.yaml
kubectl apply -f deploy/examples/pipeline-template-nodejs.yaml
kubectl apply -f deploy/examples/buildrun-manual.yaml
kubectl -n cloudivision wait --for=jsonpath='{.status.phase}'=Succeeded buildrun/demo-buildrun-manual --timeout=10m
```

The core PipelineTemplate shape is:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: PipelineTemplate
metadata: {name: app-ci, namespace: cloudivision}
spec:
  projectRef: demo-project
  steps:
    - {name: test, image: node:24-alpine, command: [npm], args: [test]}
  build: {enabled: false, builder: none, push: false}
  security: {allowPrivileged: false, runAsNonRoot: true}
```

Replace the step for common stacks:

```yaml
# Go
- {name: test, image: golang:1.26, command: [go], args: [test, ./...]}
# Java/Maven
- {name: test, image: maven:3.9-eclipse-temurin-21, command: [mvn], args: [-B, test]}
# Python
- {name: test, image: python:3.13-alpine, command: [sh], args: [-c, "pip install -r requirements.txt && pytest"]}
```

For a Dockerfile build, use rootless BuildKit infrastructure and opt in:

```yaml
build:
  enabled: true
  contextDir: .
  dockerfile: Dockerfile
  builder: buildkit
  image: ghcr.io/example/app
  push: true
```

The runner never mounts `docker.sock` and is not privileged. BuildKit must be reachable/provided and registry authentication must already be scoped to the runner. On failure inspect `status.failure`, conditions, Job Events, and logs:

```sh
kubectl -n cloudivision describe buildrun demo-buildrun-manual
kubectl -n cloudivision describe job demo-buildrun-manual-runner
kubectl -n cloudivision logs -l cloudivision.io/buildrun=demo-buildrun-manual --all-containers
```

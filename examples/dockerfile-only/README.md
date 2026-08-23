# Dockerfile-only example

A static page served by unprivileged nginx. It demonstrates a rootless BuildKit pipeline without application dependencies.

```sh
cd examples/dockerfile-only
sh test.sh
docker build -t cloudivision-static:local .
docker run --rm -p 8080:8080 cloudivision-static:local
curl http://127.0.0.1:8080/
```

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/dockerfile-only/cloudivision.yaml
kubectl -n cloudivision wait --for=jsonpath='{.status.phase}'=Succeeded buildrun/example-dockerfile-manual --timeout=10m
```

The example does not mount a Docker socket, request privileged mode, or push to a paid registry.

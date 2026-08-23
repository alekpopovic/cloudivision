# Node.js API example

This zero-dependency Node.js API uses the built-in test runner and a non-root runtime image.

```sh
cd examples/node-api
npm test
docker build -t cloudivision-node-api:local .
docker run --rm -p 8080:8080 cloudivision-node-api:local
curl http://127.0.0.1:8080/healthz
```

```sh
kubectl apply -f deploy/examples/project.yaml
kubectl apply -f examples/node-api/cloudivision.yaml
kubectl -n cloudivision wait --for=jsonpath='{.status.phase}'=Succeeded buildrun/example-node-api-manual --timeout=10m
```

No package install or paid service is required; image push is disabled.

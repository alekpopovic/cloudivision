# Go API example

A dependency-free HTTP API with a `/healthz` endpoint, unit test, multi-stage non-root image, and cloudivision resources.

```sh
cd examples/go-api
go test ./...
docker build -t cloudivision-go-api:local .
docker run --rm -p 8080:8080 cloudivision-go-api:local
curl http://127.0.0.1:8080/healthz
```

After installing cloudivision and applying `deploy/examples/project.yaml`:

```sh
kubectl apply -f examples/go-api/cloudivision.yaml
kubectl -n cloudivision wait --for=jsonpath='{.status.phase}'=Succeeded buildrun/example-go-api-manual --timeout=10m
```

The image stays local by default (`push: false`); no paid registry is required.

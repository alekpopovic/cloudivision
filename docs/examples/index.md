# Real-world examples

The `examples/` directory contains small, runnable projects plus cloudivision `PipelineTemplate`, `Repository`, and `BuildRun` resources. They use the public cloudivision repository as their source and disable registry push by default. Helm and Kustomize also include illustrative `Release` resources whose placeholder digest and Environment must be replaced before use.

Apply the shared project once:

```sh
kubectl apply -f deploy/examples/project.yaml
```

Then choose an example:

| Example | Local verification | Cluster resources |
| --- | --- | --- |
| [Go API](https://github.com/alekpopovic/cloudivision/tree/main/examples/go-api) | `cd examples/go-api && go test ./...` | `kubectl apply -f examples/go-api/cloudivision.yaml` |
| [Node.js API](https://github.com/alekpopovic/cloudivision/tree/main/examples/node-api) | `cd examples/node-api && npm test` | `kubectl apply -f examples/node-api/cloudivision.yaml` |
| [Angular app](https://github.com/alekpopovic/cloudivision/tree/main/examples/angular-app) | `cd examples/angular-app && npm ci && npm test -- --browsers=ChromeHeadless && npm run build` | `kubectl apply -f examples/angular-app/cloudivision.yaml` |
| [Python FastAPI](https://github.com/alekpopovic/cloudivision/tree/main/examples/python-fastapi) | `cd examples/python-fastapi && python -m unittest discover -s tests` | `kubectl apply -f examples/python-fastapi/cloudivision.yaml` |
| [Java Maven](https://github.com/alekpopovic/cloudivision/tree/main/examples/java-maven) | `cd examples/java-maven && mvn -B test` | `kubectl apply -f examples/java-maven/cloudivision.yaml` |
| [Dockerfile only](https://github.com/alekpopovic/cloudivision/tree/main/examples/dockerfile-only) | `cd examples/dockerfile-only && sh test.sh` | `kubectl apply -f examples/dockerfile-only/cloudivision.yaml` |
| [Helm release](https://github.com/alekpopovic/cloudivision/tree/main/examples/helm-release) | `cd examples/helm-release && sh test.sh` | `kubectl apply -f examples/helm-release/cloudivision.yaml` |
| [Kustomize release](https://github.com/alekpopovic/cloudivision/tree/main/examples/kustomize-release) | `cd examples/kustomize-release && sh test.sh` | `kubectl apply -f examples/kustomize-release/cloudivision.yaml` |

Watch any BuildRun with:

```sh
kubectl -n cloudivision get buildruns,jobs,pods
kubectl -n cloudivision logs -l cloudivision.io/buildrun=<buildrun-name> --all-containers
```

The default fixtures require only Kubernetes, public container/package sources, and optional local Docker for image smoke tests. No paid external service is required. A rootless BuildKit endpoint is needed only when executing a fixture whose `build.enabled` is `true`.

The conformance suite clones the same repository and runs the Go API unit test, preventing the evaluation example and platform test fixture from drifting apart.

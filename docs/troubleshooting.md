# Troubleshooting

Start with a compact cluster snapshot:

```sh
kubectl get crds | grep cicd.cloudivision.io
kubectl -n cloudivision get deploy,pods,serviceaccount,role,rolebinding
kubectl -n cloudivision get projects,repositories,pipelinetemplates,buildruns,environments,releases
```

## Installation pods do not become ready

```sh
kubectl -n cloudivision describe deploy
kubectl -n cloudivision get events --sort-by=.lastTimestamp
kubectl -n cloudivision logs deploy/cloudivision-cloudivision-controller --tail=200
kubectl -n cloudivision logs deploy/cloudivision-cloudivision-api --tail=200
```

For local kind installs, rebuild and load images with `./hack/kind-load-images.sh`, then rerun `./hack/install-dev.sh`.

## BuildRun remains Pending or Queued

```sh
kubectl -n cloudivision describe buildrun BUILD_RUN
kubectl -n cloudivision get job,pod -l cloudivision.io/buildrun=BUILD_RUN
kubectl -n cloudivision get events --sort-by=.lastTimestamp
```

Confirm that referenced Project, Repository and PipelineTemplate objects exist in the same namespace. Check that the Project-created runner ServiceAccount and RoleBinding exist.

## Runner fails

```sh
kubectl -n cloudivision logs -l cloudivision.io/buildrun=BUILD_RUN --all-containers --tail=200
kubectl -n cloudivision describe pod -l cloudivision.io/buildrun=BUILD_RUN
```

Private repositories need the referenced credential Secret. Image builds need rootless BuildKit tooling and registry credentials. cloudivision does not support mounting `docker.sock` or privileged Docker-in-Docker as its default path.

## API logs are unavailable

Verify the API can list pods and read `pods/log` in the BuildRun namespace. In the default chart the API Role is namespaced, so additional project namespaces need explicitly scoped API access.

## Webhook is rejected

Confirm the URL provider matches `Repository.spec.provider`, the namespace query is correct, and the Kubernetes Secret key matches the provider configuration. Signature failures intentionally do not create BuildRuns.

## UI cannot reach the API

Inspect `/assets/config.json` in the web service and set `web.config.apiBaseUrl` when the UI and API do not share an origin. Add the UI origin to `api.cors.allowedOrigins` and rerun the Helm upgrade.

## GitOps release does not progress

Inspect Release conditions and controller logs. Verify the Environment provider, repository credentials and Argo CD Application name/namespace. Missing optional Argo CD resources should be corrected in the Environment rather than granting broader controller permissions.

# Troubleshooting

Start with desired state, status, child workload, Events, then logs:

```sh
kubectl -n ci get buildrun BUILD -o yaml
kubectl -n ci get job,pod -l cloudivision.io/buildrun=BUILD
kubectl -n ci describe job BUILD-runner
kubectl -n ci logs -l cloudivision.io/buildrun=BUILD --all-containers --tail=200
kubectl -n cloudivision logs deploy/cloudivision-cloudivision-controller --tail=200
```

For a failed build, use `status.failure.reason/message` and `PolicyDenied` violations before changing configuration. `ImagePullBackOff` points to image name/auth/egress; `Forbidden` points to runner RBAC; scheduling failures point to quota/resources; clone failures point to URL/credentials/NetworkPolicy. BuildKit errors require rootless BuildKit tooling, never Docker socket access.

For Releases, inspect the Git commit/PR, `status.deployment.syncStatus` and `healthStatus`, target Environment policy, approval, provider health, and timeout. Preserve evidence before retrying as a new resource.

API errors use `{code,message,requestId,violations?}`. Search structured API logs with the request ID. Browser CORS errors require the exact origin in `api.cors.allowedOrigins`. Logs are bounded/polled, not durable artifact storage.

## Troubleshooting the troubleshooting path

If Kubernetes itself is unavailable, capture client/context errors and follow the [disaster recovery runbook](disaster-recovery.md). If an instruction appears stale, compare [configuration](../reference/configuration.md), rendered Helm output, and the current CRD schema before applying changes.

# First webhook

Create a random secret without committing it:

```sh
WEBHOOK_SECRET="$(openssl rand -hex 32)"
kubectl -n cloudivision create secret generic demo-webhook --from-literal=secret="${WEBHOOK_SECRET}"
kubectl -n cloudivision patch repository demo-repository --type=merge -p '{"spec":{"provider":"github","webhook":{"enabled":true,"secretRef":{"name":"demo-webhook","key":"secret"},"events":["push"]}}}'
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-api 8080:8080
```

The Repository configuration represented by that patch is:

```yaml
spec:
  provider: github
  webhook:
    enabled: true
    secretRef: {name: demo-webhook, key: secret}
    events: [push]
```

Expose the API through TLS ingress for a real provider and configure GitHub push delivery to `https://CI_HOST/api/v1/webhooks/github/demo-repository?namespace=cloudivision` with content type JSON and the same secret. A successful delivery returns 201; a replay returns the existing BuildRun with 200.

```sh
kubectl -n cloudivision get buildruns --sort-by=.metadata.creationTimestamp
kubectl -n cloudivision logs deploy/cloudivision-cloudivision-api --tail=100
```

Invalid signatures are rejected before parsing/creation. Rotate the Kubernetes and GitHub secret together. Do not use auth-disabled mode or an unauthenticated HTTP ingress outside local development. See [webhook security](../security/webhook-security.md).

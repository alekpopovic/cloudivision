# Configure your first webhook

Create a secret in the Repository namespace. Replace the example value before use:

```sh
kubectl -n cloudivision create secret generic demo-webhook \
  --from-literal=secret='replace-with-a-random-secret'
```

Set `Repository.spec.webhook.enabled: true` and point `secretRef` at that key:

```yaml
webhook:
  enabled: true
  secretRef:
    name: demo-webhook
    key: secret
  events:
    - push
```

Expose the API through an authenticated ingress in shared environments. For local testing only, port-forward it:

```sh
kubectl -n cloudivision port-forward svc/cloudivision-cloudivision-api 8080:8080
```

Configure the Git provider to send push events to:

```text
https://YOUR_API_HOST/api/v1/webhooks/github/REPOSITORY_NAME?namespace=cloudivision
```

Use `gitlab`, `gitea` or `generic` instead of `github` when it matches `Repository.spec.provider`. GitHub requests require `X-Hub-Signature-256`; GitLab and other providers use their provider-specific signature/token headers. cloudivision verifies the configured secret before creating a BuildRun and uses the provider event ID to suppress duplicate runs.

Check delivery and run creation:

```sh
kubectl -n cloudivision get repositories,buildruns
kubectl -n cloudivision logs deploy/cloudivision-cloudivision-api --tail=100
```

Never put the webhook secret in a Repository manifest or logs. Rotate it by updating the Kubernetes Secret and the provider configuration together.

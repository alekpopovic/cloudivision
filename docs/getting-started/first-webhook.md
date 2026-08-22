# First GitHub webhook

This guide connects one GitHub repository to the cloudivision GitHub endpoint.
The API must be reachable from GitHub over public HTTPS with a valid certificate.

## 1. Create the shared secret

Generate at least 32 random bytes in a password manager and keep the value in its
clipboard. Paste it into a silent shell prompt, then store it in a Kubernetes
Secret:

```sh
read -rsp "Webhook secret: " WEBHOOK_SECRET
echo
kubectl -n cloudivision create secret generic demo-github-webhook \
  --from-literal=secret="${WEBHOOK_SECRET}"
```

Do not print the variable, put it in a manifest, or commit it. Keep the password
manager value in the clipboard until it has also been entered in GitHub, then
clear the clipboard and run `unset WEBHOOK_SECRET`.

## 2. Enable the Repository webhook

The `Project` and `PipelineTemplate` named below must already exist:

```yaml
apiVersion: cicd.cloudivision.io/v1alpha1
kind: Repository
metadata:
  name: demo-repository
  namespace: cloudivision
spec:
  projectRef: demo-project
  provider: github
  url: https://github.com/acme/demo.git
  defaultBranch: main
  pipelineTemplateRef: app-ci
  webhook:
    enabled: true
    secretRef:
      name: demo-github-webhook
      key: secret
```

Apply it with `kubectl apply -f repository.yaml`. The API reads only the named
Secret and key; the value is never returned by the API.

## 3. Add the webhook in GitHub

In the GitHub repository:

1. Open **Settings → Webhooks → Add webhook**.
2. Set **Payload URL** to
   `https://cloudivision.example.com/api/v1/webhooks/github/demo-repository?namespace=cloudivision`.
3. Select **Content type: application/json**.
4. Paste the same password-manager value into **Secret**.
5. Keep **Enable SSL verification** selected.
6. Choose **Let me select individual events**, then select **Pushes** and
   **Pull requests**.
7. Keep **Active** selected and click **Add webhook**.
8. Return to the shell and run `unset WEBHOOK_SECRET`.

GitHub sends a signed `ping` when the hook is created. A successful delivery gets
HTTP 200 with `"event":"ping"`, `"result":"accepted"`, and no `buildRun`.

## 4. Verify a build delivery

Push to the Repository's `defaultBranch`, then inspect the delivery in GitHub's
**Recent Deliveries**. A new build returns HTTP 201 and `"result":"created"`.

```sh
kubectl -n cloudivision get buildruns
kubectl -n cloudivision get buildruns -o custom-columns=NAME:.metadata.name,EVENT:.spec.triggeredBy.eventID,COMMIT:.spec.commitSHA
```

Re-delivering the same GitHub delivery returns HTTP 200 with
`"result":"duplicate"` and does not create another BuildRun. A push whose branch
does not match `defaultBranch` returns HTTP 200 with `"result":"ignored"`.

For production idempotency across API replicas and restarts, configure the
PostgreSQL audit backend and apply all migrations in `internal/audit/migrations`.
See [webhook security](../security/webhook-security.md) for rejection behavior and
operational guidance.

# Secret handling

cloudivision resolves Kubernetes Secrets through a small provider interface. Every request names an allowed namespace, Secret and explicit required or optional keys. Resolving every key is denied unless both the request and provider explicitly opt into that behavior.

Resolved values are excluded from JSON and render as `[REDACTED SECRET]`. Controllers and the API return only presence, selected key names and safe errors. Secret values must never be written to BuildRun/Release status or logs.

The Job executor validates registry credentials through the provider and projects only the keys that validation found. An explicitly configured registry key is mounted alone. Signing keys already use a single-key Secret projection. Webhook and notification endpoints also resolve exactly one configured key.

Typical formats are:

- registry: `.dockerconfigjson`, `config.json`, or `username` plus `password`/`token`;
- webhook: the exact HMAC key named by `Repository.spec.webhook.secretRef`;
- notification: an endpoint URL under the configured Project key;
- signing: the exact private-key item named by the PipelineTemplate.

Keep each Secret in its project's namespace and grant the controller/API only `get`; do not grant list/watch for Secret values.

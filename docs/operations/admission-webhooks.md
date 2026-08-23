# Admission webhooks

cloudivision has optional mutating and validating admission webhooks for all
six v1alpha1 resources. They centralize defaults for branches, build settings,
the Job executor, production approvals, and release strategy. Validation rejects
missing webhook/GitOps dependencies, implicit privileged pipelines, and
production releases that violate digest policy.

Webhooks are disabled by default, so a simple installation has no TLS or
cert-manager dependency. To enable them with cert-manager installed:

```yaml
admissionWebhooks:
  enabled: true
  certManager:
    enabled: true
```

Alternatively set `existingSecret` to a TLS Secret containing `tls.crt` and
`tls.key`, and set `caBundle` to the base64-encoded PEM CA. Helm refuses an
incomplete TLS configuration. Keep `failurePolicy: Fail` in production and
monitor controller readiness before enabling the webhook configurations.

Privileged pipelines require both `spec.security.allowPrivileged: true` and the
annotation `cicd.cloudivision.io/allow-privileged: "true"`. This explicit opt-in
does not grant Kubernetes permissions; runner policy and Pod Security admission
still apply.

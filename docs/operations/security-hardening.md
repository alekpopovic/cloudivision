# Security hardening

Before shared or production use:

- enable OIDC and map least-privilege groups; never expose auth-disabled mode;
- terminate TLS, restrict ingress/CORS, and apply default-deny plus required egress NetworkPolicies;
- keep runner pods non-root, unprivileged, without hostPath or Docker socket, with requests/limits/deadlines;
- scope Git, registry, signing, webhook, database, and GitOps credentials per project/provider;
- require immutable digests, SBOM, signatures, vulnerability policy, and production approval where evidence is durable;
- pin component/tool images and run `make security-check` in CI;
- protect Prometheus, logs, audit data, traces, backups, and dashboard access as potentially sensitive metadata.

```yaml
namespace:
  podSecurity: {enabled: true, level: restricted}
api:
  auth:
    mode: oidc
networkPolicy: {enabled: true}
securityContext:
  allowPrivilegeEscalation: false
  privileged: false
  capabilities: {drop: [ALL]}
```

Current limitations: namespace isolation is not a hostile multi-tenant sandbox; external provider tokens are operator-managed; artifacts/evidence are not all durable; signed provenance/base-image verification is incomplete; chart NetworkPolicy egress may need environment-specific tightening; dependency advisories require continuous triage.

## Troubleshooting

On admission denial, read the namespace Pod Security labels and Event rather than weakening the cluster globally. On RBAC failure use `kubectl auth can-i --as=system:serviceaccount:NAMESPACE:ACCOUNT`. On policy denial correct the cited field/evidence. Re-run `make security-check` and document any exception with scope and expiry.

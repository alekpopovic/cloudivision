# Authentication and RBAC

The API supports development-only disabled auth and OIDC JWT validation. Roles are `admin`, `project-admin`, `developer`, and `viewer`; project-scoped mappings restrict operations to configured projects. UI hiding is not authorization—the API enforces permissions.

```yaml
api:
  auth:
    mode: oidc
    oidc: {issuerUrl: https://id.example.com, clientId: cloudivision, audience: cloudivision}
    groupMappings:
      - {group: platform-admins, role: admin, scope: global}
      - {group: storefront-devs, role: developer, scope: project, project: storefront}
```

Controller install permissions are cluster-scoped because CRDs/projects may span namespaces. Runner permissions are namespaced and should be reviewed with `kubectl auth can-i --list --as=system:serviceaccount:NS:SA -n NS`. Never grant runners cluster-admin or Secret list/watch. Rotate OIDC signing keys through issuer discovery and restrict API ingress/TLS.

Limitations: there is no built-in user/team persistence, service-account token exchange, fine-grained per-resource policy UI, or external authorization webhook. Group mapping file changes require deployment/config rollout.

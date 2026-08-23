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

The chart creates an otherwise unbound API ClusterRole permission template. The
Project controller creates a RoleBinding to that template only in each registered
project namespace. The controller can bind only that exact ClusterRole. The
central API can read project CRs and pod logs and get (but not list or watch)
named Secrets only in those namespaces. Require OIDC in production and audit
cross-project requests.

Product authorization now supports organizations, teams, memberships and
project grants. The permission order is `org-admin`, `project-admin`,
`developer`, `viewer`, and the read/audit-focused `auditor`; global `admin`
remains a platform role. A viewer cannot trigger builds, a developer can trigger
builds, and project admins can configure repositories. PostgreSQL-backed
directories use stable OIDC subjects, never mutable display names or email, as
membership identity.

Kubernetes RBAC remains an independent enforcement layer. Organization access
must never grant cross-namespace Secret access, and team labels or OIDC groups
must not be accepted as project grants without an explicit mapping. Audit events
carry an optional organization field and exports must scope it before returning
tenant data.

Limitations: management endpoints are currently read-only; invitations,
service-account token exchange, and an external authorization webhook remain
future work. Group mapping and membership changes require configuration or
database updates.

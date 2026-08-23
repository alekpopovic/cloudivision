# Organizations, teams and memberships

Organizations are the product-level tenancy boundary above Kubernetes
namespaces. Users join an organization directly or through teams. Teams receive
project grants with one of `project-admin`, `developer`, `viewer`, or `auditor`.
Organization owners use `org-admin`. The legacy global `admin` role remains for
platform administration and development auth compatibility.

Projects may set `spec.organizationRef`; an empty value means the compatibility
`default` organization. This field does not replace namespace isolation or
Kubernetes RBAC. Both product permission and Kubernetes authorization must
succeed for an operation that reaches the cluster.

The API exposes the current user's organizations and read-only team, member and
project-access views under `/api/v1/organizations`. The Angular Organization
page provides the switcher and those views. Mutation workflows are intentionally
deferred until identity lifecycle and invitation semantics are defined.

With PostgreSQL audit storage, operators can apply
`internal/auth/migrations/0001_organizations.sql` and the API uses those tables
in authenticated mode. Development/auth-disabled mode uses an in-memory
`default` organization with the `dev-user` as org admin so existing local flows
continue to work.

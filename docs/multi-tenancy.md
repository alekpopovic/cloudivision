# Multi-tenancy and runner security

cloudivision projects can request namespace isolation with `Project.spec.isolation.createNamespace=true`.
The Project controller creates the namespace, Pod Security Admission labels, a runner ServiceAccount,
namespaced Role/RoleBinding, and optional NetworkPolicies.

`networkPolicyMode=egressAllowList` currently creates a deny-by-default egress policy skeleton. The
Project API intentionally does not yet contain CIDR, namespace selector, DNS name, or port allow-list
fields. Add those fields to `ProjectIsolation` before opening egress for tenant workloads.

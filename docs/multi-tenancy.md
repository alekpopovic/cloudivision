# Multi-tenancy and runner security

cloudivision projects can request namespace isolation with `Project.spec.isolation.createNamespace=true`.
The Project controller creates the namespace, Pod Security Admission labels, a runner ServiceAccount,
namespaced Role/RoleBinding, and optional NetworkPolicies.

For the current Kubernetes Job executor, `Project.spec.namespace` must match the namespace where the
Project, Repository, PipelineTemplate and BuildRun CRs live. Kubernetes ownerReferences for namespaced
objects cannot safely cross namespaces, and the runner ServiceAccount/RBAC is namespaced. The
BuildRun controller fails fast when this invariant is violated instead of creating a Job that cannot
use the intended tenant RBAC.

`networkPolicyMode=egressAllowList` currently creates a deny-by-default egress policy skeleton. The
Project API intentionally does not yet contain CIDR, namespace selector, DNS name, or port allow-list
fields. Add those fields to `ProjectIsolation` before opening egress for tenant workloads.

# 69. Organizations, Teams and Memberships

```text
Continue working on the cloudivision project.

Task:
Add product-level organization/team model.

Goal:
Prepare cloudivision for real multi-user usage.

Model:
- Organization
- Team
- Membership
- RoleBinding or internal permission binding

Storage:
- PostgreSQL if auth/audit DB exists
- or CRD-backed model if product direction prefers Kubernetes-native

Roles:
- org-admin
- project-admin
- developer
- viewer
- auditor

Behavior:
1. Users belong to organizations.
2. Projects belong to organizations.
3. Teams can be granted project access.
4. API permissions use organization/team membership.
5. Audit events include organization.

Angular UI:
- Organization switcher
- Team management page
- Member list
- Project access page

Docs:
- docs/concepts/organizations.md
- docs/security/auth-rbac.md

Tests:
- membership role checks
- project access checks
- viewer cannot trigger build
- developer can trigger build
- project-admin can configure repo

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Permission model supports organizations.
- Existing single-user/dev mode still works.
```

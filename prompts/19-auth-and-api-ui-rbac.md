# Prompt 19 — Auth and API/UI RBAC

Phase: Auth

```text
Continue working on the cloudivision project.

Task:
Add the initial auth model for cloudivision API and Angular UI.

Goal:
Implement a clear and secure foundation without fake security.

Auth modes:
- disabled
- oidc

Config:
- CLOU_DIVISION_AUTH_MODE=disabled|oidc
- CLOU_DIVISION_OIDC_ISSUER_URL
- CLOU_DIVISION_OIDC_CLIENT_ID
- CLOU_DIVISION_OIDC_AUDIENCE optional
- CLOU_DIVISION_OIDC_JWKS_URL optional

Steps:
1. Implement auth middleware:
   - disabled allows request, actor=dev-user and marks development mode.
   - oidc validates JWT through issuer/JWKS.
2. Add Principal model: subject, email, groups, displayName.
3. Add permission model: role admin/project-admin/developer/viewer; scope global/project.
4. MVP group mapping through config YAML/ConfigMap.
5. Endpoint protection: viewer GET, developer trigger BuildRun, project-admin create/update project resources, admin everything.
6. Audit events use principal actor.
7. Angular UI shows current user and hides actions user cannot perform, but backend remains source of truth.
8. Angular API interceptor attaches bearer token if auth service provides one and handles 401/403 clearly.
9. Tests: disabled mode, invalid/valid OIDC token with test JWKS, permission matrix, Angular guard/UI permission helper tests.

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Auth disabled is explicitly development mode.
- OIDC mode rejects requests without valid token.
- Permission checks exist in API handlers/middleware.
- No hardcoded admin tokens.
- UI does not treat frontend hiding as security.
```

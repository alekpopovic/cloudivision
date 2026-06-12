package auth

import (
	"net/http"
	"strings"
)

type Permission string

const (
	PermissionRead           Permission = "read"
	PermissionTriggerBuild   Permission = "trigger-build"
	PermissionManageProjects Permission = "manage-projects"
	PermissionApproveRelease Permission = "approve-release"
	PermissionAdmin          Permission = "admin"
)

func RequiredPermission(method, path string) (Permission, bool) {
	if path == "/healthz" || path == "/readyz" {
		return PermissionRead, false
	}
	if strings.HasPrefix(path, "/api/v1/webhooks/") {
		return PermissionTriggerBuild, false
	}
	if method == http.MethodGet {
		return PermissionRead, true
	}
	if method == http.MethodPost && path == "/api/v1/build-runs" {
		return PermissionTriggerBuild, true
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/v1/releases/") && (strings.HasSuffix(path, "/approve") || strings.HasSuffix(path, "/reject")) {
		return PermissionApproveRelease, true
	}
	if method == http.MethodPost && (path == "/api/v1/projects" || path == "/api/v1/repositories" || path == "/api/v1/pipeline-templates") {
		return PermissionManageProjects, true
	}
	return PermissionAdmin, true
}

func Allowed(principal *Principal, permission Permission) bool {
	if principal == nil {
		return false
	}
	for _, role := range principal.Roles {
		if role == RoleAdmin {
			return true
		}
		switch permission {
		case PermissionRead:
			if role == RoleViewer || role == RoleDeveloper || role == RoleProjectAdmin {
				return true
			}
		case PermissionTriggerBuild, PermissionApproveRelease:
			if role == RoleDeveloper || role == RoleProjectAdmin {
				return true
			}
		case PermissionManageProjects:
			if role == RoleProjectAdmin {
				return true
			}
		}
	}
	return false
}

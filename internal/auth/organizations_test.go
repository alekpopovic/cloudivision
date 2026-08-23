package auth

import (
	"context"
	"testing"
)

func TestOrganizationProjectPermissions(t *testing.T) {
	directory := &MemoryDirectory{Organizations: []Organization{{ID: "acme", Name: "Acme"}}, Teams: []Team{{ID: "app", OrganizationID: "acme", Name: "App"}}, Memberships: []Membership{{OrganizationID: "acme", UserSubject: "viewer", TeamID: "app", Role: RoleViewer}, {OrganizationID: "acme", UserSubject: "developer", TeamID: "app", Role: RoleDeveloper}, {OrganizationID: "acme", UserSubject: "owner", TeamID: "app", Role: RoleProjectAdmin}}, ProjectAccess: []ProjectAccess{{OrganizationID: "acme", Project: "store", TeamID: "app", Role: RoleViewer}}}
	// Change each grant to assert the role matrix independently.
	if ok, _ := directory.Allowed(context.Background(), &Principal{Subject: "viewer"}, "acme", "store", PermissionTriggerBuild); ok {
		t.Fatal("viewer can trigger build")
	}
	directory.ProjectAccess[0].Role = RoleDeveloper
	if ok, _ := directory.Allowed(context.Background(), &Principal{Subject: "developer"}, "acme", "store", PermissionTriggerBuild); !ok {
		t.Fatal("developer cannot trigger build")
	}
	directory.ProjectAccess[0].Role = RoleProjectAdmin
	if ok, _ := directory.Allowed(context.Background(), &Principal{Subject: "owner"}, "acme", "store", PermissionManageProjects); !ok {
		t.Fatal("project-admin cannot configure repository")
	}
}

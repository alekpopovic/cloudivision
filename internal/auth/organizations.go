package auth

import (
	"context"
	"errors"
	"sync"
)

var ErrOrganizationForbidden = errors.New("organization access denied")

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Team struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
}
type Membership struct {
	OrganizationID string `json:"organizationId"`
	UserSubject    string `json:"userSubject"`
	TeamID         string `json:"teamId,omitempty"`
	Role           Role   `json:"role"`
}
type ProjectAccess struct {
	OrganizationID string `json:"organizationId"`
	Project        string `json:"project"`
	TeamID         string `json:"teamId"`
	Role           Role   `json:"role"`
}

type OrganizationDirectory interface {
	ListOrganizations(context.Context, string) ([]Organization, error)
	ListTeams(context.Context, string) ([]Team, error)
	ListMemberships(context.Context, string) ([]Membership, error)
	ListProjectAccess(context.Context, string) ([]ProjectAccess, error)
	Allowed(context.Context, *Principal, string, string, Permission) (bool, error)
}

type MemoryDirectory struct {
	mu            sync.RWMutex
	Organizations []Organization
	Teams         []Team
	Memberships   []Membership
	ProjectAccess []ProjectAccess
}

func NewDevelopmentDirectory() *MemoryDirectory {
	return &MemoryDirectory{Organizations: []Organization{{ID: "default", Name: "Default organization"}}, Teams: []Team{{ID: "platform", OrganizationID: "default", Name: "Platform"}}, Memberships: []Membership{{OrganizationID: "default", UserSubject: "dev-user", Role: RoleOrgAdmin}, {OrganizationID: "default", UserSubject: "dev-user", TeamID: "platform", Role: RoleDeveloper}}}
}

func (d *MemoryDirectory) ListOrganizations(_ context.Context, subject string) ([]Organization, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	allowed := map[string]bool{}
	for _, m := range d.Memberships {
		if m.UserSubject == subject {
			allowed[m.OrganizationID] = true
		}
	}
	out := []Organization{}
	for _, o := range d.Organizations {
		if allowed[o.ID] {
			out = append(out, o)
		}
	}
	return out, nil
}
func (d *MemoryDirectory) ListTeams(_ context.Context, organization string) ([]Team, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []Team{}
	for _, v := range d.Teams {
		if v.OrganizationID == organization {
			out = append(out, v)
		}
	}
	return out, nil
}
func (d *MemoryDirectory) ListMemberships(_ context.Context, organization string) ([]Membership, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []Membership{}
	for _, v := range d.Memberships {
		if v.OrganizationID == organization {
			out = append(out, v)
		}
	}
	return out, nil
}
func (d *MemoryDirectory) ListProjectAccess(_ context.Context, organization string) ([]ProjectAccess, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []ProjectAccess{}
	for _, v := range d.ProjectAccess {
		if v.OrganizationID == organization {
			out = append(out, v)
		}
	}
	return out, nil
}

func (d *MemoryDirectory) Allowed(_ context.Context, principal *Principal, organization, project string, permission Permission) (bool, error) {
	if principal == nil {
		return false, nil
	}
	if Allowed(principal, PermissionAdmin) {
		return true, nil
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	teams := map[string]bool{}
	for _, membership := range d.Memberships {
		if membership.OrganizationID != organization || membership.UserSubject != principal.Subject {
			continue
		}
		if membership.TeamID != "" {
			teams[membership.TeamID] = true
			continue
		}
		if Allowed(&Principal{Roles: []Role{membership.Role}}, permission) {
			return true, nil
		}
	}
	if project == "" {
		return false, nil
	}
	for _, access := range d.ProjectAccess {
		if access.OrganizationID == organization && access.Project == project && teams[access.TeamID] && Allowed(&Principal{Roles: []Role{access.Role}}, permission) {
			return true, nil
		}
	}
	return false, nil
}

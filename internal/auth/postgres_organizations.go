package auth

import (
	"context"
	"database/sql"
	"fmt"
)

type PostgresDirectory struct{ DB *sql.DB }

func (d PostgresDirectory) ListOrganizations(ctx context.Context, subject string) ([]Organization, error) {
	rows, err := d.DB.QueryContext(ctx, `select o.id,o.name from organizations o join memberships m on m.organization_id=o.id where m.user_subject=$1 group by o.id,o.name order by o.name`, subject)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}
	defer rows.Close()
	out := []Organization{}
	for rows.Next() {
		var v Organization
		if err := rows.Scan(&v.ID, &v.Name); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (d PostgresDirectory) ListTeams(ctx context.Context, organization string) ([]Team, error) {
	rows, err := d.DB.QueryContext(ctx, `select id,organization_id,name from teams where organization_id=$1 order by name`, organization)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()
	out := []Team{}
	for rows.Next() {
		var v Team
		if err := rows.Scan(&v.ID, &v.OrganizationID, &v.Name); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (d PostgresDirectory) ListMemberships(ctx context.Context, organization string) ([]Membership, error) {
	rows, err := d.DB.QueryContext(ctx, `select organization_id,user_subject,coalesce(team_id,''),role from memberships where organization_id=$1 order by user_subject`, organization)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()
	out := []Membership{}
	for rows.Next() {
		var v Membership
		if err := rows.Scan(&v.OrganizationID, &v.UserSubject, &v.TeamID, &v.Role); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (d PostgresDirectory) ListProjectAccess(ctx context.Context, organization string) ([]ProjectAccess, error) {
	rows, err := d.DB.QueryContext(ctx, `select organization_id,project,team_id,role from project_access where organization_id=$1 order by project,team_id`, organization)
	if err != nil {
		return nil, fmt.Errorf("list project access: %w", err)
	}
	defer rows.Close()
	out := []ProjectAccess{}
	for rows.Next() {
		var v ProjectAccess
		if err := rows.Scan(&v.OrganizationID, &v.Project, &v.TeamID, &v.Role); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (d PostgresDirectory) Allowed(ctx context.Context, principal *Principal, organization, project string, permission Permission) (bool, error) {
	memberships, err := d.ListMemberships(ctx, organization)
	if err != nil {
		return false, err
	}
	access, err := d.ListProjectAccess(ctx, organization)
	if err != nil {
		return false, err
	}
	memory := &MemoryDirectory{Memberships: memberships, ProjectAccess: access}
	return memory.Allowed(ctx, principal, organization, project, permission)
}

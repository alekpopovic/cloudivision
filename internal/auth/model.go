package auth

import (
	"context"
	"net/http"
)

type Role string

const (
	RoleAdmin        Role = "admin"
	RoleProjectAdmin Role = "project-admin"
	RoleDeveloper    Role = "developer"
	RoleViewer       Role = "viewer"
)

type Scope string

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

type Principal struct {
	Subject     string   `json:"subject"`
	Email       string   `json:"email,omitempty"`
	Groups      []string `json:"groups,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Roles       []Role   `json:"roles,omitempty"`
	DevMode     bool     `json:"devMode,omitempty"`
}

type Authenticator interface {
	Authenticate(r *http.Request) (*Principal, error)
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal *Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(*Principal)
	return principal, ok && principal != nil
}

func ActorFromContext(ctx context.Context, fallback string) string {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || principal.Subject == "" {
		return fallback
	}
	if principal.Email != "" {
		return principal.Email
	}
	return principal.Subject
}

type DisabledAuthenticator struct{}

func (DisabledAuthenticator) Authenticate(*http.Request) (*Principal, error) {
	return &Principal{
		Subject:     "dev-user",
		Email:       "dev-user@localhost",
		DisplayName: "Development User",
		Roles:       []Role{RoleAdmin},
		DevMode:     true,
	}, nil
}

package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDisabledAuthenticatorReturnsDevAdmin(t *testing.T) {
	principal, err := DisabledAuthenticator{}.Authenticate(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Subject != "dev-user" || !principal.DevMode {
		t.Fatalf("principal = %#v, want dev-user development principal", principal)
	}
	if !Allowed(principal, PermissionAdmin) {
		t.Fatalf("disabled principal should have admin permissions in development mode")
	}
}

func TestPermissionMatrix(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		want       bool
	}{
		{name: "viewer reads", role: RoleViewer, permission: PermissionRead, want: true},
		{name: "viewer cannot trigger", role: RoleViewer, permission: PermissionTriggerBuild, want: false},
		{name: "developer triggers", role: RoleDeveloper, permission: PermissionTriggerBuild, want: true},
		{name: "developer cannot manage projects", role: RoleDeveloper, permission: PermissionManageProjects, want: false},
		{name: "project admin manages projects", role: RoleProjectAdmin, permission: PermissionManageProjects, want: true},
		{name: "admin does everything", role: RoleAdmin, permission: PermissionAdmin, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Allowed(&Principal{Roles: []Role{tt.role}}, tt.permission)
			if got != tt.want {
				t.Fatalf("Allowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRolesForGroups(t *testing.T) {
	roles := RolesForGroups([]string{"team-devs", "team-viewers"}, GroupMappingConfig{Mappings: []GroupMapping{
		{Group: "team-devs", Role: RoleDeveloper, Scope: ScopeProject, Project: "demo"},
		{Group: "team-admins", Role: RoleAdmin, Scope: ScopeGlobal},
	}})
	if len(roles) != 1 || roles[0] != RoleDeveloper {
		t.Fatalf("roles = %#v, want developer", roles)
	}
}

func TestOIDCAuthenticatorValidatesToken(t *testing.T) {
	key := testRSAKey(t)
	now := time.Unix(1_700_000_000, 0)
	jwksData, err := json.Marshal(testJWKS(key.PublicKey, "test-key"))
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	authenticator, err := NewOIDCAuthenticator(OIDCConfig{
		IssuerURL: "https://issuer.example",
		ClientID:  "cloudivision",
		JWKSURL:   "https://issuer.example/jwks",
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(string(jwksData))),
				Header:     make(http.Header),
			}, nil
		})},
		Now: func() time.Time { return now },
		Groups: GroupMappingConfig{Mappings: []GroupMapping{
			{Group: "devs", Role: RoleDeveloper, Scope: ScopeProject, Project: "demo"},
		}},
	})
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}
	token := signTestJWT(t, key, "test-key", map[string]any{
		"iss":    "https://issuer.example",
		"sub":    "user-1",
		"aud":    "cloudivision",
		"exp":    now.Add(time.Hour).Unix(),
		"email":  "user@example.com",
		"name":   "User One",
		"groups": []string{"devs"},
	})

	principal, err := authenticator.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if principal.Subject != "user-1" || principal.Email != "user@example.com" {
		t.Fatalf("principal = %#v", principal)
	}
	if len(principal.Roles) != 1 || principal.Roles[0] != RoleDeveloper {
		t.Fatalf("roles = %#v, want developer", principal.Roles)
	}
}

func TestOIDCAuthenticatorRejectsInvalidToken(t *testing.T) {
	authenticator, err := NewOIDCAuthenticator(OIDCConfig{IssuerURL: "https://issuer.example", ClientID: "cloudivision"})
	if err != nil {
		t.Fatalf("NewOIDCAuthenticator() error = %v", err)
	}
	if _, err := authenticator.ValidateToken(context.Background(), "not-a-jwt"); err == nil {
		t.Fatal("ValidateToken() error = nil, want error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func signTestJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	header := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid}
	encodedHeader := encodeJSONPart(t, header)
	encodedClaims := encodeJSONPart(t, claims)
	signed := encodedHeader + "." + encodedClaims
	digest := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func encodeJSONPart(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal jwt part: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func testJWKS(key rsa.PublicKey, kid string) map[string]any {
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
			},
		},
	}
}

func TestRequiredPermission(t *testing.T) {
	permission, protected := RequiredPermission(http.MethodPost, "/api/v1/build-runs")
	if !protected || permission != PermissionTriggerBuild {
		t.Fatalf("build-runs permission = %q protected=%v", permission, protected)
	}
	_, protected = RequiredPermission(http.MethodPost, "/api/v1/webhooks/github/repo")
	if protected {
		t.Fatal("webhooks should not require bearer auth because they use provider signatures")
	}
}

func ExampleGroupMappingConfig() {
	config := GroupMappingConfig{Mappings: []GroupMapping{{Group: "platform", Role: RoleAdmin, Scope: ScopeGlobal}}}
	fmt.Println(RolesForGroups([]string{"platform"}, config)[0])
	// Output: admin
}

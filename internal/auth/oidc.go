package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrMissingBearerToken = errors.New("missing bearer token")
	ErrInvalidToken       = errors.New("invalid token")
)

type OIDCConfig struct {
	IssuerURL string
	ClientID  string
	Audience  string
	JWKSURL   string
	Groups    GroupMappingConfig
	Client    *http.Client
	Now       func() time.Time
}

type OIDCAuthenticator struct {
	config OIDCConfig
	mu     sync.Mutex
	keys   map[string]*rsa.PublicKey
}

func NewOIDCAuthenticator(config OIDCConfig) (*OIDCAuthenticator, error) {
	if strings.TrimSpace(config.IssuerURL) == "" {
		return nil, fmt.Errorf("CLOU_DIVISION_OIDC_ISSUER_URL is required when auth mode is oidc")
	}
	if strings.TrimSpace(config.ClientID) == "" {
		return nil, fmt.Errorf("CLOU_DIVISION_OIDC_CLIENT_ID is required when auth mode is oidc")
	}
	if config.JWKSURL == "" {
		config.JWKSURL = strings.TrimRight(config.IssuerURL, "/") + "/.well-known/jwks.json"
	}
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &OIDCAuthenticator{config: config}, nil
}

func (a *OIDCAuthenticator) Authenticate(r *http.Request) (*Principal, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, ErrMissingBearerToken
	}
	return a.ValidateToken(r.Context(), strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
}

func (a *OIDCAuthenticator) ValidateToken(ctx context.Context, token string) (*Principal, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}
	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return nil, fmt.Errorf("%w: decode header: %v", ErrInvalidToken, err)
	}
	if header.Alg != "RS256" || header.KID == "" {
		return nil, fmt.Errorf("%w: only RS256 tokens with kid are supported", ErrInvalidToken)
	}
	key, err := a.key(ctx, header.KID)
	if err != nil {
		return nil, err
	}
	signed := []byte(parts[0] + "." + parts[1])
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: decode signature: %v", ErrInvalidToken, err)
	}
	digest := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return nil, fmt.Errorf("%w: verify signature: %v", ErrInvalidToken, err)
	}
	var claims jwtClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return nil, fmt.Errorf("%w: decode claims: %v", ErrInvalidToken, err)
	}
	if err := a.validateClaims(claims); err != nil {
		return nil, err
	}
	return &Principal{
		Subject:     claims.Subject,
		Email:       claims.Email,
		Groups:      claims.Groups,
		DisplayName: claims.DisplayName(),
		Roles:       RolesForGroups(claims.Groups, a.config.Groups),
	}, nil
}

func (a *OIDCAuthenticator) validateClaims(claims jwtClaims) error {
	now := a.config.Now().Unix()
	if claims.Issuer != a.config.IssuerURL {
		return fmt.Errorf("%w: issuer mismatch", ErrInvalidToken)
	}
	if claims.Subject == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidToken)
	}
	if claims.ExpiresAt == 0 || claims.ExpiresAt <= now {
		return fmt.Errorf("%w: token is expired", ErrInvalidToken)
	}
	if claims.NotBefore != 0 && claims.NotBefore > now {
		return fmt.Errorf("%w: token is not valid yet", ErrInvalidToken)
	}
	audience := a.config.Audience
	if audience == "" {
		audience = a.config.ClientID
	}
	if !claims.HasAudience(audience) {
		return fmt.Errorf("%w: audience mismatch", ErrInvalidToken)
	}
	return nil
}

func (a *OIDCAuthenticator) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	a.mu.Lock()
	if a.keys != nil {
		key := a.keys[kid]
		a.mu.Unlock()
		if key != nil {
			return key, nil
		}
	} else {
		a.mu.Unlock()
	}
	keys, err := a.fetchJWKS(ctx)
	if err != nil {
		return nil, err
	}
	key := keys[kid]
	if key == nil {
		return nil, fmt.Errorf("%w: jwks key %q not found", ErrInvalidToken, kid)
	}
	a.mu.Lock()
	a.keys = keys
	a.mu.Unlock()
	return key, nil
}

func (a *OIDCAuthenticator) fetchJWKS(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.config.JWKSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := a.config.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("fetch JWKS: status %d", resp.StatusCode)
	}
	var set jwks
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	keys := map[string]*rsa.PublicKey{}
	for _, item := range set.Keys {
		if item.KID == "" || item.KTY != "RSA" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(item.N)
		if err != nil {
			return nil, fmt.Errorf("decode jwk modulus: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(item.E)
		if err != nil {
			return nil, fmt.Errorf("decode jwk exponent: %w", err)
		}
		exponent := new(big.Int).SetBytes(eBytes).Int64()
		keys[item.KID] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent)}
	}
	return keys, nil
}

func decodeJWTPart(part string, out any) error {
	data, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

type jwtHeader struct {
	Alg string `json:"alg"`
	KID string `json:"kid"`
}

type jwtClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  audience `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	NotBefore int64    `json:"nbf,omitempty"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Groups    []string `json:"groups,omitempty"`
}

func (c jwtClaims) HasAudience(expected string) bool {
	for _, item := range c.Audience {
		if item == expected {
			return true
		}
	}
	return false
}

func (c jwtClaims) DisplayName() string {
	if c.Name != "" {
		return c.Name
	}
	if c.Email != "" {
		return c.Email
	}
	return c.Subject
}

type audience []string

func (a *audience) UnmarshalJSON(data []byte) error {
	var list []string
	if err := json.Unmarshal(data, &list); err == nil {
		*a = list
		return nil
	}
	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*a = []string{single}
	return nil
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

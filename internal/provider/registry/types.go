package registry

import (
	"context"
	"errors"
	"net/http"
	"time"

	coreprovider "github.com/cloudivision/cloudivision/internal/provider"
)

var (
	ErrCredentialsMissing  = errors.New("registry credentials are missing")
	ErrCredentialsInvalid  = errors.New("registry credentials are invalid")
	ErrUnsupportedProvider = errors.New("registry provider is not implemented")
	ErrImageInvalid        = errors.New("registry image reference is invalid")
)

type RegistryProvider interface {
	Name() string
	Login(ctx context.Context, req LoginRequest) (*LoginResult, error)
	ResolveImage(ctx context.Context, req ImageRequest) (*ImageRef, error)
	ReadDigest(ctx context.Context, req ImageRequest) (string, error)
	HealthCheck(ctx context.Context) coreprovider.ProviderHealth
}

type Credential struct {
	Username         string
	Password         string
	Token            string
	DockerConfigJSON []byte
}

func (c Credential) Empty() bool {
	return c.Username == "" && c.Password == "" && c.Token == "" && len(c.DockerConfigJSON) == 0
}

type LoginRequest struct {
	Registry   string
	Credential Credential
}

type LoginResult struct {
	Registry         string
	DockerConfigJSON []byte
	ExpiresAt        *time.Time
}

type ImageRequest struct {
	Registry    string
	ImagePrefix string
	Repository  string
	Tag         string
	Digest      string
	Credential  Credential
}

type ImageRef struct {
	Repository string
	Tag        string
	Digest     string
}

func (r ImageRef) String() string {
	if r.Digest != "" {
		return r.Repository + "@" + r.Digest
	}
	if r.Tag != "" {
		return r.Repository + ":" + r.Tag
	}
	return r.Repository
}

type Adapter struct {
	ProviderName    string
	DefaultRegistry string
	Implemented     bool
	Client          *http.Client
}

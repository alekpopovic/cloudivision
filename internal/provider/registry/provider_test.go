package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const providerTestDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func TestParseCredentialFormats(t *testing.T) {
	for _, test := range []struct {
		name string
		data map[string][]byte
		want Credential
	}{
		{name: "username password", data: map[string][]byte{UsernameKey: []byte("robot"), PasswordKey: []byte("secret")}, want: Credential{Username: "robot", Password: "secret"}},
		{name: "token", data: map[string][]byte{TokenKey: []byte("token-value")}, want: Credential{Token: "token-value"}},
		{name: "docker config", data: map[string][]byte{DockerConfigJSONKey: []byte(`{"auths":{"registry.example":{"auth":"cm9ib3Q6c2VjcmV0"}}}`)}, want: Credential{DockerConfigJSON: []byte(`{"auths":{"registry.example":{"auth":"cm9ib3Q6c2VjcmV0"}}}`)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseCredential(test.data)
			if err != nil {
				t.Fatalf("ParseCredential() error = %v", err)
			}
			if got.Username != test.want.Username || got.Password != test.want.Password || got.Token != test.want.Token || string(got.DockerConfigJSON) != string(test.want.DockerConfigJSON) {
				t.Fatalf("ParseCredential() = %#v", got)
			}
		})
	}
}

func TestParseCredentialRejectsMissingAndInvalid(t *testing.T) {
	if _, err := ParseCredential(nil); !errors.Is(err, ErrCredentialsMissing) {
		t.Fatalf("missing credentials error = %v", err)
	}
	if _, err := ParseCredential(map[string][]byte{PasswordKey: []byte("secret")}); !errors.Is(err, ErrCredentialsInvalid) {
		t.Fatalf("password without username error = %v", err)
	}
	if _, err := ParseCredential(map[string][]byte{DockerConfigJSONKey: []byte(`{"auths":{}}`)}); !errors.Is(err, ErrCredentialsInvalid) {
		t.Fatalf("empty docker config error = %v", err)
	}
}

func TestGenericLoginBuildsDockerConfigWithoutChangingCredential(t *testing.T) {
	credential := Credential{Username: "robot", Password: "very-secret"}
	result, err := Generic().Login(context.Background(), LoginRequest{Registry: "registry.example", Credential: credential})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(result.DockerConfigJSON, &config); err != nil {
		t.Fatal(err)
	}
	wantAuth := base64.StdEncoding.EncodeToString([]byte("robot:very-secret"))
	if config.Auths["registry.example"].Auth != wantAuth {
		t.Fatalf("docker auth = %q, want %q", config.Auths["registry.example"].Auth, wantAuth)
	}
	masked := Redactor(credential).Mask("password=very-secret auth=" + wantAuth)
	if strings.Contains(masked, "very-secret") || strings.Contains(masked, wantAuth) {
		t.Fatalf("redactor leaked credential: %q", masked)
	}
}

func TestDockerConfigPassesThroughAndWritesMode0600(t *testing.T) {
	config := []byte(`{"auths":{"registry.example":{"auth":"cm9ib3Q6c2VjcmV0"}}}`)
	result, err := Generic().Login(context.Background(), LoginRequest{
		Registry: "registry.example", Credential: Credential{DockerConfigJSON: config},
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "docker")
	path, err := WriteDockerConfig(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestResolveImageUsesPrefixAndDigest(t *testing.T) {
	image, err := GHCR().ResolveImage(context.Background(), ImageRequest{
		ImagePrefix: "ghcr.io/example", Repository: "api", Tag: "main", Digest: providerTestDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if image.Repository != "ghcr.io/example/api" || image.String() != "ghcr.io/example/api@"+providerTestDigest {
		t.Fatalf("image = %#v (%s)", image, image.String())
	}
}

func TestReadDigestUsesScopedAuthorization(t *testing.T) {
	var authorization string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		authorization = r.Header.Get("Authorization")
		if r.Method != http.MethodHead || r.URL.Path != "/v2/team/app/manifests/main" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Docker-Content-Digest": []string{providerTestDigest}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})}
	provider := Generic()
	provider.Client = client
	digest, err := provider.ReadDigest(context.Background(), ImageRequest{
		Registry: "https://registry.test", Repository: "registry.test/team/app", Tag: "main",
		Credential: Credential{Username: "robot", Token: "token-value"},
	})
	if err != nil {
		t.Fatalf("ReadDigest() error = %v", err)
	}
	if digest != providerTestDigest {
		t.Fatalf("digest = %q", digest)
	}
	if !strings.HasPrefix(authorization, "Basic ") || strings.Contains(authorization, "token-value") {
		t.Fatalf("authorization header is not scoped Basic auth: %q", authorization)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestUnsupportedProviderAndSkeletonHealth(t *testing.T) {
	if _, err := New("unknown"); !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("New() error = %v", err)
	}
	ecr := ECR()
	if health := ecr.HealthCheck(context.Background()); health.Healthy || !strings.Contains(health.Message, "skeleton") {
		t.Fatalf("ECR health = %#v", health)
	}
	if _, err := ecr.Login(context.Background(), LoginRequest{}); !errors.Is(err, ErrUnsupportedProvider) {
		t.Fatalf("ECR Login() error = %v", err)
	}
}

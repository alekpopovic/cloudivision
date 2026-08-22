package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var registryDigestPattern = regexp.MustCompile(`^sha256:[a-fA-F0-9]{64}$`)

func (p *Adapter) ResolveImage(_ context.Context, req ImageRequest) (*ImageRef, error) {
	if p == nil || !p.Implemented {
		return nil, fmt.Errorf("registry provider is unavailable: %w", ErrUnsupportedProvider)
	}
	repository := strings.Trim(strings.TrimSpace(req.Repository), "/")
	prefix := strings.Trim(strings.TrimSpace(req.ImagePrefix), "/")
	if repository == "" {
		return nil, fmt.Errorf("repository is required: %w", ErrImageInvalid)
	}
	if strings.ContainsAny(repository, " \t\r\n@") || repository == "." || repository == ".." || strings.Contains(repository, "/../") {
		return nil, fmt.Errorf("repository %q is invalid: %w", repository, ErrImageInvalid)
	}
	if prefix != "" && !hasRegistryHost(repository) && repository != prefix && !strings.HasPrefix(repository, prefix+"/") {
		repository = prefix + "/" + repository
	}
	if !hasRegistryHost(repository) && p.DefaultRegistry != "" {
		repository = p.DefaultRegistry + "/" + repository
	}
	digest := strings.ToLower(strings.TrimSpace(req.Digest))
	if digest != "" && !registryDigestPattern.MatchString(digest) {
		return nil, fmt.Errorf("digest %q is invalid: %w", req.Digest, ErrImageInvalid)
	}
	tag := strings.TrimSpace(req.Tag)
	if strings.ContainsAny(tag, "@/ \t\r\n") {
		return nil, fmt.Errorf("tag %q is invalid: %w", tag, ErrImageInvalid)
	}
	return &ImageRef{Repository: repository, Tag: tag, Digest: digest}, nil
}

func (p *Adapter) ReadDigest(ctx context.Context, req ImageRequest) (string, error) {
	image, err := p.ResolveImage(ctx, req)
	if err != nil {
		return "", err
	}
	if image.Digest != "" {
		return image.Digest, nil
	}
	reference := image.Tag
	if reference == "" {
		return "", fmt.Errorf("tag or digest is required: %w", ErrImageInvalid)
	}
	registryURL, repositoryPath, err := registryEndpoint(req.Registry, image.Repository)
	if err != nil {
		return "", err
	}
	manifestURL := strings.TrimSuffix(registryURL, "/") + "/v2/" + repositoryPath + "/manifests/" + url.PathEscape(reference)
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, manifestURL, nil)
	if err != nil {
		return "", fmt.Errorf("create registry manifest request: %w", err)
	}
	request.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))
	applyCredential(request, req.Credential, registryURL)
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("read registry manifest digest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("registry manifest request returned HTTP %d", response.StatusCode)
	}
	digest := strings.ToLower(strings.TrimSpace(response.Header.Get("Docker-Content-Digest")))
	if !registryDigestPattern.MatchString(digest) {
		return "", fmt.Errorf("registry returned an invalid Docker-Content-Digest header: %w", ErrImageInvalid)
	}
	return digest, nil
}

func registryEndpoint(configured, repository string) (string, string, error) {
	first, rest, ok := strings.Cut(repository, "/")
	if !ok || !hasRegistryHost(repository) {
		return "", "", fmt.Errorf("repository %q has no registry host: %w", repository, ErrImageInvalid)
	}
	endpoint := strings.TrimSpace(configured)
	if endpoint == "" {
		endpoint = "https://" + first
	} else if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return "", "", fmt.Errorf("registry endpoint %q is invalid: %w", configured, ErrImageInvalid)
	}
	return strings.TrimSuffix(endpoint, "/"), rest, nil
}

func RegistryHost(repository string) (string, error) {
	first, _, ok := strings.Cut(repository, "/")
	if !ok || !hasRegistryHost(repository) {
		return "", fmt.Errorf("repository %q has no registry host: %w", repository, ErrImageInvalid)
	}
	return first, nil
}

func hasRegistryHost(repository string) bool {
	first, _, ok := strings.Cut(repository, "/")
	return ok && (strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost")
}

func applyCredential(request *http.Request, credential Credential, registryURL string) {
	if credential.Token != "" && credential.Username == "" {
		request.Header.Set("Authorization", "Bearer "+credential.Token)
		return
	}
	if credential.Username != "" {
		secret := credential.Password
		if credential.Token != "" {
			secret = credential.Token
		}
		request.SetBasicAuth(credential.Username, secret)
		return
	}
	if len(credential.DockerConfigJSON) == 0 {
		return
	}
	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if json.Unmarshal(credential.DockerConfigJSON, &config) != nil {
		return
	}
	parsed, _ := url.Parse(registryURL)
	for _, key := range []string{parsed.Host, registryURL, strings.TrimPrefix(registryURL, "https://"), strings.TrimPrefix(registryURL, "http://")} {
		entry, ok := config.Auths[key]
		if !ok || entry.Auth == "" {
			continue
		}
		if _, err := base64.StdEncoding.DecodeString(entry.Auth); err == nil {
			request.Header.Set("Authorization", "Basic "+entry.Auth)
		}
		return
	}
}

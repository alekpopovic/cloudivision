package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

type Provider string

const (
	ProviderGitHub  Provider = "github"
	ProviderGitLab  Provider = "gitlab"
	ProviderGitea   Provider = "gitea"
	ProviderGeneric Provider = "generic"
)

func Verify(provider Provider, headers http.Header, body []byte, secret string) error {
	if secret == "" {
		return fmt.Errorf("webhook secret is empty")
	}
	switch provider {
	case ProviderGitHub:
		return verifyHMACHeader(headers.Get("X-Hub-Signature-256"), "sha256=", body, secret)
	case ProviderGitLab:
		if !hmac.Equal([]byte(headers.Get("X-Gitlab-Token")), []byte(secret)) {
			return fmt.Errorf("invalid GitLab webhook token")
		}
		return nil
	case ProviderGitea:
		signature := headers.Get("X-Gitea-Signature")
		if signature != "" {
			return verifyHMACHeader(signature, "", body, secret)
		}
		if !hmac.Equal([]byte(headers.Get("X-Gitea-Token")), []byte(secret)) {
			return fmt.Errorf("invalid Gitea webhook token")
		}
		return nil
	case ProviderGeneric:
		bearer := strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")
		token := headers.Get("X-Cloudivision-Token")
		if !hmac.Equal([]byte(bearer), []byte(secret)) && !hmac.Equal([]byte(token), []byte(secret)) {
			return fmt.Errorf("invalid generic webhook token")
		}
		return nil
	default:
		return fmt.Errorf("unsupported webhook provider %q", provider)
	}
}

func verifyHMACHeader(header, prefix string, body []byte, secret string) error {
	if header == "" {
		return fmt.Errorf("missing webhook signature")
	}
	signature := strings.TrimPrefix(header, prefix)
	actual, err := hex.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("invalid webhook signature encoding")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := mac.Sum(nil)
	if !hmac.Equal(actual, expected) {
		return fmt.Errorf("invalid webhook signature")
	}
	return nil
}

func SignGitHub(body []byte, secret string) string {
	return "sha256=" + signHex(body, secret)
}

func SignHex(body []byte, secret string) string {
	return signHex(body, secret)
}

func signHex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

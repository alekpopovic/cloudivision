package webhook

import (
	"net/http"
	"testing"
)

func TestVerifyGitHubSignature(t *testing.T) {
	body := []byte(`{"zen":"Keep it logically awesome."}`)
	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", SignGitHub(body, "secret"))
	if err := Verify(ProviderGitHub, headers, body, "secret"); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyGitHubSignatureRejectsInvalidSignature(t *testing.T) {
	body := []byte(`{"zen":"Keep it logically awesome."}`)
	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", SignGitHub(body, "wrong"))
	if err := Verify(ProviderGitHub, headers, body, "secret"); err == nil {
		t.Fatal("Verify() error = nil, want error")
	}
}

func TestVerifyGitLabToken(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Gitlab-Token", "secret")
	if err := Verify(ProviderGitLab, headers, []byte("{}"), "secret"); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

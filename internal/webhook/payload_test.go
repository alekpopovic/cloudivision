package webhook

import (
	"net/http"
	"testing"
)

func TestParseGitHubPullRequest(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-GitHub-Event", "pull_request")
	headers.Set("X-GitHub-Delivery", "delivery-pr-1")
	event, err := Parse(ProviderGitHub, headers, []byte(`{
  "action":"synchronize",
  "pull_request":{
    "updated_at":"2026-08-22T19:00:00Z",
    "head":{"ref":"feature/payments","sha":"abcdef0123456789"},
    "base":{"ref":"main"}
  },
  "repository":{"clone_url":"https://github.com/acme/payments.git"},
  "sender":{"login":"octocat"}
}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !event.IsPullRequest || event.Type != EventPullRequest || event.Action != "synchronize" {
		t.Fatalf("event type = %#v", event)
	}
	if event.Branch != "feature/payments" || event.BaseBranch != "main" || event.CommitSHA != "abcdef0123456789" {
		t.Fatalf("event refs = %#v", event)
	}
}

func TestParseGitHubPing(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-GitHub-Event", "ping")
	headers.Set("X-GitHub-Delivery", "delivery-ping-1")
	event, err := Parse(ProviderGitHub, headers, []byte(`{"zen":"Keep it logically awesome.","sender":{"login":"octocat"}}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !event.IsPing || event.Type != EventPing || event.EventID != "delivery-ping-1" {
		t.Fatalf("event = %#v", event)
	}
}

func TestParseGitHubTag(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-GitHub-Event", "push")
	event, err := Parse(ProviderGitHub, headers, []byte(`{"ref":"refs/tags/v1.2.0","after":"abcdef"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !event.IsPush || !event.IsTag || event.Branch != "v1.2.0" {
		t.Fatalf("event = %#v", event)
	}
}

func TestParseGitHubForkPullRequest(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-GitHub-Event", "pull_request")
	event, err := Parse(ProviderGitHub, headers, []byte(`{"action":"opened","pull_request":{"head":{"ref":"change","sha":"abcdef","repo":{"full_name":"contributor/repo"}},"base":{"ref":"main"}},"repository":{"full_name":"acme/repo"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !event.IsPullRequest || !event.IsFork {
		t.Fatalf("event = %#v", event)
	}
}

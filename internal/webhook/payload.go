package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Event struct {
	RepositoryURL string
	Branch        string
	CommitSHA     string
	Actor         string
	EventID       string
	IsPush        bool
}

func Parse(provider Provider, headers http.Header, body []byte) (Event, error) {
	switch provider {
	case ProviderGitHub:
		return parseGitHub(headers, body)
	case ProviderGitLab:
		return parseGitLab(headers, body)
	case ProviderGitea:
		return parseGitea(headers, body)
	case ProviderGeneric:
		return parseGeneric(headers, body)
	default:
		return Event{}, fmt.Errorf("unsupported webhook provider %q", provider)
	}
}

func parseGitHub(headers http.Header, body []byte) (Event, error) {
	var payload struct {
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Repository struct {
			CloneURL string `json:"clone_url"`
			HTMLURL  string `json:"html_url"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("parse GitHub payload: %w", err)
	}
	return Event{
		RepositoryURL: firstNonEmpty(payload.Repository.CloneURL, payload.Repository.HTMLURL),
		Branch:        branchFromRef(payload.Ref),
		CommitSHA:     payload.After,
		Actor:         payload.Sender.Login,
		EventID:       headers.Get("X-GitHub-Delivery"),
		IsPush:        headers.Get("X-GitHub-Event") == "push",
	}, nil
}

func parseGitLab(headers http.Header, body []byte) (Event, error) {
	var payload struct {
		ObjectKind   string `json:"object_kind"`
		Ref          string `json:"ref"`
		CheckoutSHA  string `json:"checkout_sha"`
		After        string `json:"after"`
		UserName     string `json:"user_name"`
		UserUsername string `json:"user_username"`
		Project      struct {
			GitHTTPURL string `json:"git_http_url"`
			WebURL     string `json:"web_url"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("parse GitLab payload: %w", err)
	}
	return Event{
		RepositoryURL: firstNonEmpty(payload.Project.GitHTTPURL, payload.Project.WebURL),
		Branch:        branchFromRef(payload.Ref),
		CommitSHA:     firstNonEmpty(payload.CheckoutSHA, payload.After),
		Actor:         firstNonEmpty(payload.UserUsername, payload.UserName),
		EventID:       firstNonEmpty(headers.Get("X-Gitlab-Event-UUID"), headers.Get("X-Gitlab-Event")),
		IsPush:        payload.ObjectKind == "push",
	}, nil
}

func parseGitea(headers http.Header, body []byte) (Event, error) {
	var payload struct {
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Repository struct {
			CloneURL string `json:"clone_url"`
			HTMLURL  string `json:"html_url"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("parse Gitea payload: %w", err)
	}
	return Event{
		RepositoryURL: firstNonEmpty(payload.Repository.CloneURL, payload.Repository.HTMLURL),
		Branch:        branchFromRef(payload.Ref),
		CommitSHA:     payload.After,
		Actor:         payload.Sender.Login,
		EventID:       firstNonEmpty(headers.Get("X-Gitea-Delivery"), headers.Get("X-Gitea-Event")),
		IsPush:        headers.Get("X-Gitea-Event") == "push",
	}, nil
}

func parseGeneric(headers http.Header, body []byte) (Event, error) {
	var payload struct {
		RepositoryURL string `json:"repositoryURL"`
		Branch        string `json:"branch"`
		CommitSHA     string `json:"commitSHA"`
		Actor         string `json:"actor"`
		EventID       string `json:"eventID"`
		EventType     string `json:"eventType"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("parse generic payload: %w", err)
	}
	return Event{
		RepositoryURL: payload.RepositoryURL,
		Branch:        payload.Branch,
		CommitSHA:     payload.CommitSHA,
		Actor:         payload.Actor,
		EventID:       firstNonEmpty(payload.EventID, headers.Get("X-Cloudivision-Event-ID")),
		IsPush:        payload.EventType == "" || payload.EventType == "push",
	}, nil
}

func branchFromRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

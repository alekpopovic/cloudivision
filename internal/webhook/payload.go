package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type EventType string

const (
	EventPush        EventType = "push"
	EventPullRequest EventType = "pull_request"
	EventPing        EventType = "ping"
	EventUnknown     EventType = "unknown"
)

type Event struct {
	RepositoryURL string
	Branch        string
	BaseBranch    string
	CommitSHA     string
	Actor         string
	EventID       string
	Type          EventType
	Action        string
	IsTag         bool
	IsFork        bool
	TrustedActor  bool
	Timestamp     time.Time
	IsPush        bool
	IsPullRequest bool
	IsPing        bool
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

func DeliveryID(provider Provider, headers http.Header) string {
	switch provider {
	case ProviderGitHub:
		return headers.Get("X-GitHub-Delivery")
	case ProviderGitLab:
		return firstNonEmpty(headers.Get("X-Gitlab-Event-UUID"), headers.Get("X-Gitlab-Event"))
	case ProviderGitea:
		return firstNonEmpty(headers.Get("X-Gitea-Delivery"), headers.Get("X-Gitea-Event"))
	case ProviderGeneric:
		return headers.Get("X-Cloudivision-Event-ID")
	default:
		return ""
	}
}

func parseGitHub(headers http.Header, body []byte) (Event, error) {
	eventType := EventType(headers.Get("X-GitHub-Event"))
	if eventType == "" {
		eventType = EventUnknown
	}
	var payload struct {
		Ref         string `json:"ref"`
		After       string `json:"after"`
		Action      string `json:"action"`
		PullRequest struct {
			UpdatedAt time.Time `json:"updated_at"`
			Head      struct {
				Ref  string `json:"ref"`
				SHA  string `json:"sha"`
				Repo struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
			} `json:"base"`
		} `json:"pull_request"`
		Repository struct {
			CloneURL string `json:"clone_url"`
			HTMLURL  string `json:"html_url"`
			FullName string `json:"full_name"`
			PushedAt int64  `json:"pushed_at"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("parse GitHub payload: %w", err)
	}
	isTag := strings.HasPrefix(payload.Ref, "refs/tags/")
	branch := refName(payload.Ref)
	commitSHA := payload.After
	baseBranch := ""
	var timestamp time.Time
	if payload.Repository.PushedAt > 0 {
		timestamp = time.Unix(payload.Repository.PushedAt, 0).UTC()
	}
	if eventType == EventPullRequest {
		branch = payload.PullRequest.Head.Ref
		baseBranch = payload.PullRequest.Base.Ref
		commitSHA = payload.PullRequest.Head.SHA
		timestamp = payload.PullRequest.UpdatedAt
	}
	isFork := eventType == EventPullRequest && payload.PullRequest.Head.Repo.FullName != "" && payload.Repository.FullName != "" && !strings.EqualFold(payload.PullRequest.Head.Repo.FullName, payload.Repository.FullName)
	return Event{
		RepositoryURL: firstNonEmpty(payload.Repository.CloneURL, payload.Repository.HTMLURL),
		Branch:        branch,
		BaseBranch:    baseBranch,
		CommitSHA:     commitSHA,
		Actor:         payload.Sender.Login,
		EventID:       headers.Get("X-GitHub-Delivery"),
		Type:          eventType,
		Action:        payload.Action,
		IsTag:         isTag,
		IsFork:        isFork,
		Timestamp:     timestamp,
		IsPush:        eventType == EventPush,
		IsPullRequest: eventType == EventPullRequest,
		IsPing:        eventType == EventPing,
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
	isPush := payload.ObjectKind == "push"
	return Event{
		RepositoryURL: firstNonEmpty(payload.Project.GitHTTPURL, payload.Project.WebURL),
		Branch:        refName(payload.Ref),
		IsTag:         strings.HasPrefix(payload.Ref, "refs/tags/"),
		CommitSHA:     firstNonEmpty(payload.CheckoutSHA, payload.After),
		Actor:         firstNonEmpty(payload.UserUsername, payload.UserName),
		EventID:       firstNonEmpty(headers.Get("X-Gitlab-Event-UUID"), headers.Get("X-Gitlab-Event")),
		Type:          eventTypeForPush(isPush),
		IsPush:        isPush,
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
	isPush := headers.Get("X-Gitea-Event") == "push"
	return Event{
		RepositoryURL: firstNonEmpty(payload.Repository.CloneURL, payload.Repository.HTMLURL),
		Branch:        refName(payload.Ref),
		IsTag:         strings.HasPrefix(payload.Ref, "refs/tags/"),
		CommitSHA:     payload.After,
		Actor:         payload.Sender.Login,
		EventID:       firstNonEmpty(headers.Get("X-Gitea-Delivery"), headers.Get("X-Gitea-Event")),
		Type:          eventTypeForPush(isPush),
		IsPush:        isPush,
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
		Tag           string `json:"tag"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Event{}, fmt.Errorf("parse generic payload: %w", err)
	}
	isTag := payload.EventType == "tag" || payload.Tag != ""
	isPush := payload.EventType == "" || payload.EventType == "push" || isTag
	branch := payload.Branch
	if isTag {
		branch = firstNonEmpty(payload.Tag, payload.Branch)
	}
	return Event{
		RepositoryURL: payload.RepositoryURL,
		Branch:        branch,
		IsTag:         isTag,
		CommitSHA:     payload.CommitSHA,
		Actor:         payload.Actor,
		EventID:       firstNonEmpty(payload.EventID, headers.Get("X-Cloudivision-Event-ID")),
		Type:          eventTypeForPush(isPush),
		IsPush:        isPush,
	}, nil
}

func eventTypeForPush(isPush bool) EventType {
	if isPush {
		return EventPush
	}
	return EventUnknown
}

func refName(ref string) string {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	return strings.TrimPrefix(ref, "refs/tags/")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

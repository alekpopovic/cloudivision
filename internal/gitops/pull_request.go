package gitops

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrPullRequestProviderNotConfigured = errors.New("pull request provider is not configured")
	ErrPullRequestProviderUnsupported   = errors.New("pull request provider is unsupported")
)

type PullRequestProvider interface {
	CreateOrUpdatePullRequest(ctx context.Context, req PullRequestRequest) (*PullRequestResult, error)
	ReadPullRequestStatus(ctx context.Context, req PullRequestStatusRequest) (*PullRequestResult, error)
}

type PullRequestRequest struct {
	RepositoryURL string
	ReleaseName   string
	HeadBranch    string
	TargetBranch  string
	Title         string
	Body          string
	Reviewers     []string
	Labels        []string
}

type PullRequestStatusRequest struct {
	RepositoryURL string
	Reference     string
}

type PullRequestResult struct {
	Provider     string
	URL          string
	Reference    string
	HeadBranch   string
	TargetBranch string
	MergeStatus  string
}

// PullRequestAPI is implemented by provider-specific authenticated API clients.
type PullRequestAPI interface {
	CreateOrUpdate(ctx context.Context, provider string, req PullRequestRequest) (*PullRequestResult, error)
	ReadStatus(ctx context.Context, provider string, req PullRequestStatusRequest) (*PullRequestResult, error)
}

type GitHubPullRequestProvider struct{ API PullRequestAPI }

func (p GitHubPullRequestProvider) CreateOrUpdatePullRequest(ctx context.Context, req PullRequestRequest) (*PullRequestResult, error) {
	if p.API == nil {
		return nil, fmt.Errorf("github: %w; configure an authenticated GitHub API client", ErrPullRequestProviderNotConfigured)
	}
	return p.API.CreateOrUpdate(ctx, "github", req)
}

func (p GitHubPullRequestProvider) ReadPullRequestStatus(ctx context.Context, req PullRequestStatusRequest) (*PullRequestResult, error) {
	if p.API == nil {
		return nil, fmt.Errorf("github: %w; configure an authenticated GitHub API client", ErrPullRequestProviderNotConfigured)
	}
	return p.API.ReadStatus(ctx, "github", req)
}

type GitLabPullRequestProvider struct{ API PullRequestAPI }

func (p GitLabPullRequestProvider) CreateOrUpdatePullRequest(ctx context.Context, req PullRequestRequest) (*PullRequestResult, error) {
	if p.API == nil {
		return nil, fmt.Errorf("gitlab: %w; configure an authenticated GitLab API client", ErrPullRequestProviderNotConfigured)
	}
	return p.API.CreateOrUpdate(ctx, "gitlab", req)
}

func (p GitLabPullRequestProvider) ReadPullRequestStatus(ctx context.Context, req PullRequestStatusRequest) (*PullRequestResult, error) {
	if p.API == nil {
		return nil, fmt.Errorf("gitlab: %w; configure an authenticated GitLab API client", ErrPullRequestProviderNotConfigured)
	}
	return p.API.ReadStatus(ctx, "gitlab", req)
}

type UnsupportedPullRequestProvider struct{ RepositoryURL string }

func (p UnsupportedPullRequestProvider) CreateOrUpdatePullRequest(context.Context, PullRequestRequest) (*PullRequestResult, error) {
	return nil, fmt.Errorf("repository %q: %w; use GitHub or GitLab, or configure a custom provider", p.RepositoryURL, ErrPullRequestProviderUnsupported)
}

func (p UnsupportedPullRequestProvider) ReadPullRequestStatus(context.Context, PullRequestStatusRequest) (*PullRequestResult, error) {
	return nil, fmt.Errorf("repository %q: %w; use GitHub or GitLab, or configure a custom provider", p.RepositoryURL, ErrPullRequestProviderUnsupported)
}

func PullRequestProviderForRepository(repositoryURL string) PullRequestProvider {
	lower := strings.ToLower(repositoryURL)
	switch {
	case strings.Contains(lower, "github.com"):
		return GitHubPullRequestProvider{}
	case strings.Contains(lower, "gitlab.com"):
		return GitLabPullRequestProvider{}
	default:
		return UnsupportedPullRequestProvider{RepositoryURL: repositoryURL}
	}
}

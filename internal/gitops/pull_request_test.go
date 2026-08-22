package gitops

import (
	"context"
	"errors"
	"testing"
)

func TestPullRequestProviderSkeletonsReturnClearConfigurationErrors(t *testing.T) {
	providers := []PullRequestProvider{GitHubPullRequestProvider{}, GitLabPullRequestProvider{}}
	for _, provider := range providers {
		_, err := provider.CreateOrUpdatePullRequest(context.Background(), PullRequestRequest{})
		if !errors.Is(err, ErrPullRequestProviderNotConfigured) {
			t.Fatalf("CreateOrUpdatePullRequest() error = %v, want not configured", err)
		}
	}
}

func TestPullRequestProviderForRepositoryRejectsGenericHost(t *testing.T) {
	provider := PullRequestProviderForRepository("https://git.example.com/platform/deployments.git")
	_, err := provider.CreateOrUpdatePullRequest(context.Background(), PullRequestRequest{})
	if !errors.Is(err, ErrPullRequestProviderUnsupported) {
		t.Fatalf("CreateOrUpdatePullRequest() error = %v, want unsupported", err)
	}
}

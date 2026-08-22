package webhook

import (
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

func TestFilter(t *testing.T) {
	repository := &cicdv1alpha1.Repository{Spec: cicdv1alpha1.RepositorySpec{
		DefaultBranch: "main",
		Webhook: cicdv1alpha1.RepositoryWebhook{
			BranchFilters: cicdv1alpha1.RepositoryRefFilters{Include: []string{"main", "release/*"}, Exclude: []string{"release/private-*"}},
			TagFilters:    cicdv1alpha1.RepositoryRefFilters{Include: []string{"v*"}, Exclude: []string{"v0.*"}},
			PullRequest:   cicdv1alpha1.RepositoryPullRequestFilters{Enabled: true, Events: []string{"opened", "synchronize"}},
		},
	}}
	tests := []struct {
		name   string
		event  Event
		want   bool
		reason string
	}{
		{"included branch", Event{IsPush: true, Type: EventPush, Branch: "release/1.2"}, true, "branch_matched"},
		{"excluded branch", Event{IsPush: true, Type: EventPush, Branch: "release/private-fix"}, false, "branch_ignored"},
		{"tag", Event{IsPush: true, IsTag: true, Type: EventPush, Branch: "v1.2.0"}, true, "tag_matched"},
		{"pull request", Event{IsPullRequest: true, Type: EventPullRequest, Action: "synchronize", BaseBranch: "main"}, true, "pull_request_matched"},
		{"fork pull request", Event{IsPullRequest: true, Type: EventPullRequest, Action: "opened", BaseBranch: "main", IsFork: true}, false, "fork_pull_request_blocked"},
		{"unknown", Event{Type: EventUnknown}, false, "event_ignored"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason, _ := Filter(repository, tt.event)
			if got != tt.want || reason != tt.reason {
				t.Fatalf("Filter() = %v, %q, want %v, %q", got, reason, tt.want, tt.reason)
			}
		})
	}
}

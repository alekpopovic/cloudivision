package webhook

import (
	"path"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

// Filter evaluates a verified event against Repository webhook configuration.
// The returned reason is stable audit metadata; message is safe for callers.
func Filter(repository *cicdv1alpha1.Repository, event Event) (bool, string, string) {
	config := repository.Spec.Webhook
	if event.IsPing {
		return true, "ping", ""
	}
	if !event.IsPush && !event.IsPullRequest {
		return false, "event_ignored", "Webhook event is not configured to create a BuildRun."
	}
	if event.IsPullRequest {
		if !config.PullRequest.Enabled {
			return false, "pull_request_disabled", "Pull request builds are disabled for this repository."
		}
		events := config.PullRequest.Events
		if len(events) == 0 {
			events = []string{"opened", "reopened", "synchronize"}
		}
		if !matchesAny(event.Action, events) {
			return false, "pull_request_action_ignored", "Pull request action is not configured to create a BuildRun."
		}
		if event.IsFork && !config.PullRequest.BuildForks {
			return false, "fork_pull_request_blocked", "Pull request builds from forks are disabled for this repository."
		}
		if config.PullRequest.RequireTrustedActor && !event.TrustedActor {
			return false, "untrusted_actor", "Pull request actor could not be verified as trusted."
		}
		if !matchesRef(event.BaseBranch, config.BranchFilters, []string{repository.Spec.DefaultBranch}) {
			return false, "branch_ignored", "Pull request base branch does not match repository branch filters."
		}
		return true, "pull_request_matched", ""
	}
	if event.IsTag {
		if len(config.TagFilters.Include) == 0 {
			return false, "tag_disabled", "Tag builds are disabled until tagFilters.include is configured."
		}
		if !matchesRef(event.Branch, config.TagFilters, nil) {
			return false, "tag_ignored", "Tag does not match repository tag filters."
		}
		return true, "tag_matched", ""
	}
	if !matchesRef(event.Branch, config.BranchFilters, []string{repository.Spec.DefaultBranch}) {
		return false, "branch_ignored", "Push branch does not match repository branch filters."
	}
	return true, "branch_matched", ""
}

func matchesRef(value string, filters cicdv1alpha1.RepositoryRefFilters, fallbackInclude []string) bool {
	if matchesAny(value, filters.Exclude) {
		return false
	}
	include := filters.Include
	if len(include) == 0 {
		include = fallbackInclude
	}
	return len(include) > 0 && matchesAny(value, include)
}

func matchesAny(value string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == value {
			return true
		}
		if matched, err := path.Match(pattern, value); err == nil && matched {
			return true
		}
	}
	return false
}

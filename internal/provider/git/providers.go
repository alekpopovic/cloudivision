package git

import "github.com/cloudivision/cloudivision/internal/provider"

func Generic() provider.Provider {
	return provider.Static{ProviderName: "generic", ProviderType: "git", Healthy: true, Message: "generic Git CLI adapter is available", Features: []provider.Capability{{Name: "clone", Description: "Clone repositories over supported Git transports"}, {Name: "checkout", Description: "Check out branches, tags, or commits"}}}
}
func GitHub() provider.Provider {
	return provider.Static{ProviderName: "github", ProviderType: "git", Healthy: true, Message: "GitHub webhook and repository skeleton is registered", Features: []provider.Capability{{Name: "webhooks", Description: "Verify and parse GitHub webhooks"}, {Name: "pull-requests", Description: "Pull request adapter skeleton; credentials required"}}}
}
func GitLab() provider.Provider {
	return provider.Static{ProviderName: "gitlab", ProviderType: "git", Healthy: true, Message: "GitLab webhook and repository skeleton is registered", Features: []provider.Capability{{Name: "webhooks", Description: "Verify and parse GitLab webhooks"}, {Name: "merge-requests", Description: "Merge request adapter skeleton; credentials required"}}}
}

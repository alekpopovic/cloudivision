package notifications

import "github.com/cloudivision/cloudivision/internal/provider"

func Noop() provider.Provider {
	return provider.Static{ProviderName: "noop", ProviderType: "notifications", Healthy: true, Message: "notifications are disabled", Features: []provider.Capability{{Name: "discard", Description: "Accept notification events without external delivery"}}}
}

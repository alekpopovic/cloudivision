package registry

import "github.com/cloudivision/cloudivision/internal/provider"

func Generic() provider.Provider {
	return provider.Static{ProviderName: "generic", ProviderType: "registry", Healthy: true, Message: "OCI-compatible registry configuration is available", Features: []provider.Capability{{Name: "push", Description: "Push OCI images through configured build adapters"}, {Name: "digest", Description: "Record immutable image digests"}}}
}

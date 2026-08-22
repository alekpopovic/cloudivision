package secrets

import "github.com/cloudivision/cloudivision/internal/provider"

func Kubernetes() provider.Provider {
	return provider.Static{ProviderName: "kubernetes", ProviderType: "secrets", Healthy: true, Message: "Kubernetes Secret projection adapter is available", Features: []provider.Capability{{Name: "secret-projection", Description: "Project named Secret keys into workloads"}}}
}

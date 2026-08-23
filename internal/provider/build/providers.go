package build

import "github.com/cloudivision/cloudivision/internal/provider"

func BuildKit() provider.Provider {
	return provider.Static{ProviderName: "buildkit", ProviderType: "build", Binary: "buildctl", Features: []provider.Capability{{Name: "rootless-image-build", Description: "Build and push OCI images without a Docker socket"}}}
}

func RemoteAgent() provider.Provider {
	return provider.Static{ProviderName: "remote-agent", ProviderType: "build", Healthy: false, Message: "experimental architecture skeleton; execution and registration are disabled", Features: []provider.Capability{{Name: "pull-based-runner", Description: "Planned short-lived, project-scoped remote runner assignments"}}}
}

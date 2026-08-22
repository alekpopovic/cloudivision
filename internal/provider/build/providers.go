package build

import "github.com/cloudivision/cloudivision/internal/provider"

func BuildKit() provider.Provider {
	return provider.Static{ProviderName: "buildkit", ProviderType: "build", Binary: "buildctl", Features: []provider.Capability{{Name: "rootless-image-build", Description: "Build and push OCI images without a Docker socket"}}}
}

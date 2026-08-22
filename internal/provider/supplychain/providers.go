package supplychain

import "github.com/cloudivision/cloudivision/internal/provider"

func Noop() provider.Provider {
	return provider.Static{ProviderName: "noop", ProviderType: "supply-chain", Healthy: true, Message: "noop supply-chain hooks are the safe default", Features: []provider.Capability{{Name: "sbom", Description: "Noop SBOM hook"}, {Name: "scan", Description: "Noop scan hook"}, {Name: "sign", Description: "Noop signing hook"}, {Name: "provenance", Description: "Noop provenance hook"}}}
}
func Syft() provider.Provider {
	return provider.Static{ProviderName: "syft", ProviderType: "supply-chain", Binary: "syft", Features: []provider.Capability{{Name: "sbom", Description: "Generate SPDX JSON SBOMs"}}}
}
func Grype() provider.Provider {
	return provider.Static{ProviderName: "grype", ProviderType: "supply-chain", Binary: "grype", Features: []provider.Capability{{Name: "scan", Description: "Scan images and summarize vulnerabilities"}}}
}
func Cosign() provider.Provider {
	return provider.Static{ProviderName: "cosign", ProviderType: "supply-chain", Binary: "cosign", Features: []provider.Capability{{Name: "sign", Description: "Sign images with explicit keyless or key-based mode"}}}
}

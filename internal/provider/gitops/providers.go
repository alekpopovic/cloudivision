package gitops

import "github.com/cloudivision/cloudivision/internal/provider"

func Generic() provider.Provider {
	return provider.Static{ProviderName: "generic", ProviderType: "gitops", Healthy: true, Message: "generic commit-based GitOps adapter is available", Features: []provider.Capability{{Name: "update-image", Description: "Commit desired image changes"}}}
}
func ArgoCD() provider.Provider {
	return provider.Static{ProviderName: "argocd", ProviderType: "gitops", Healthy: true, Message: "Argo CD status adapter is registered; per-application RBAC is checked at use time", Features: []provider.Capability{{Name: "sync-status", Description: "Read Argo CD sync status"}, {Name: "health-status", Description: "Read Argo CD health independently"}}}
}

package gitops

import (
	"context"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

// ProviderStatusReader routes status reads without requiring provider CRDs in the scheme.
type ProviderStatusReader struct {
	ArgoCD StatusReader
	Flux   StatusReader
}

func (r ProviderStatusReader) ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error) {
	switch req.Provider {
	case cicdv1alpha1.GitOpsProviderArgoCD:
		return r.ArgoCD.ReadDeploymentStatus(ctx, req)
	case cicdv1alpha1.GitOpsProviderFlux:
		return r.Flux.ReadDeploymentStatus(ctx, req)
	default:
		return nil, ErrDeploymentStatusUnavailable
	}
}

package gitops

import (
	"context"
	"fmt"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ArgoCDStatusReader struct {
	Client client.Client
}

func (r ArgoCDStatusReader) ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error) {
	if req.Provider != cicdv1alpha1.GitOpsProviderArgoCD {
		return nil, ErrDeploymentStatusUnavailable
	}
	if r.Client == nil {
		return nil, fmt.Errorf("read Argo CD status: %w", ErrProviderUnavailable)
	}
	if req.ApplicationName == "" || req.Namespace == "" {
		return nil, fmt.Errorf("read Argo CD status: application name and namespace are required")
	}
	application := &unstructured.Unstructured{}
	application.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Application",
	})
	if err := r.Client.Get(ctx, types.NamespacedName{Name: req.ApplicationName, Namespace: req.Namespace}, application); err != nil {
		if meta.IsNoMatchError(err) {
			return nil, ErrProviderUnavailable
		}
		if apierrors.IsNotFound(err) {
			return nil, ErrDeploymentResourceMissing
		}
		return nil, fmt.Errorf("read Argo CD Application %s/%s: %w", req.Namespace, req.ApplicationName, err)
	}
	syncStatus, _, _ := unstructured.NestedString(application.Object, "status", "sync", "status")
	healthStatus, _, _ := unstructured.NestedString(application.Object, "status", "health", "status")
	operationPhase, _, _ := unstructured.NestedString(application.Object, "status", "operationState", "phase")
	reconciledAt, _, _ := unstructured.NestedString(application.Object, "status", "reconciledAt")
	revision, _, _ := unstructured.NestedString(application.Object, "status", "sync", "revision")
	return &DeploymentStatus{SyncStatus: syncStatus, HealthStatus: healthStatus, OperationPhase: operationPhase, ObservedRevision: revision, ObservedAt: reconciledAt}, nil
}

package gitops

import (
	"context"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FluxStatusReader struct{ Client client.Client }

func (r FluxStatusReader) ReadDeploymentStatus(ctx context.Context, req DeploymentStatusRequest) (*DeploymentStatus, error) {
	if r.Client == nil {
		return nil, fmt.Errorf("read Flux status: %w", ErrProviderUnavailable)
	}
	kind := req.ResourceKind
	if kind == "" {
		kind = "Kustomization"
	}
	var gvk schema.GroupVersionKind
	switch strings.ToLower(kind) {
	case "kustomization":
		gvk = schema.GroupVersionKind{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Kind: "Kustomization"}
	case "helmrelease":
		gvk = schema.GroupVersionKind{Group: "helm.toolkit.fluxcd.io", Version: "v2", Kind: "HelmRelease"}
	default:
		return nil, fmt.Errorf("unsupported Flux resource kind %q", kind)
	}
	resource := &unstructured.Unstructured{}
	resource.SetGroupVersionKind(gvk)
	if err := r.Client.Get(ctx, types.NamespacedName{Name: req.ApplicationName, Namespace: req.Namespace}, resource); err != nil {
		if meta.IsNoMatchError(err) {
			return nil, ErrProviderUnavailable
		}
		if apierrors.IsNotFound(err) {
			return nil, ErrDeploymentResourceMissing
		}
		return nil, fmt.Errorf("read Flux %s %s/%s: %w", kind, req.Namespace, req.ApplicationName, err)
	}
	ready := conditionStatus(resource, "Ready")
	revision, _, _ := unstructured.NestedString(resource.Object, "status", "lastAppliedRevision")
	if revision == "" {
		revision, _, _ = unstructured.NestedString(resource.Object, "status", "lastAttemptedRevision")
	}
	observedAt := conditionTime(resource, "Ready")
	syncStatus, healthStatus := "Unknown", "Unknown"
	switch ready {
	case "True":
		syncStatus, healthStatus = "Synced", "Healthy"
	case "False":
		syncStatus, healthStatus = "OutOfSync", "Degraded"
	}
	return &DeploymentStatus{SyncStatus: syncStatus, HealthStatus: healthStatus, ObservedRevision: revision, ObservedAt: observedAt}, nil
}

func conditionStatus(resource *unstructured.Unstructured, conditionType string) string {
	conditions, _, _ := unstructured.NestedSlice(resource.Object, "status", "conditions")
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if condition["type"] == conditionType {
			value, _ := condition["status"].(string)
			return value
		}
	}
	return ""
}

func conditionTime(resource *unstructured.Unstructured, conditionType string) string {
	conditions, _, _ := unstructured.NestedSlice(resource.Object, "status", "conditions")
	for _, item := range conditions {
		condition, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if condition["type"] == conditionType {
			value, _ := condition["lastTransitionTime"].(string)
			return value
		}
	}
	return ""
}

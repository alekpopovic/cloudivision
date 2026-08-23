package gitops

import (
	"context"
	"errors"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFluxStatusReaderReady(t *testing.T) {
	resource := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1", "kind": "Kustomization",
		"metadata": map[string]any{"name": "app", "namespace": "flux-system"},
		"status":   map[string]any{"lastAppliedRevision": "main@sha1:abc", "conditions": []any{map[string]any{"type": "Ready", "status": "True", "lastTransitionTime": "2026-08-23T00:00:00Z"}}},
	}}
	reader := FluxStatusReader{Client: fake.NewClientBuilder().WithRuntimeObjects(resource).Build()}
	status, err := reader.ReadDeploymentStatus(context.Background(), DeploymentStatusRequest{Provider: cicdv1alpha1.GitOpsProviderFlux, ApplicationName: "app", Namespace: "flux-system", ResourceKind: "Kustomization"})
	if err != nil {
		t.Fatal(err)
	}
	if status.SyncStatus != "Synced" || status.HealthStatus != "Healthy" || status.ObservedRevision != "main@sha1:abc" {
		t.Fatalf("status = %#v", status)
	}
}

func TestFluxStatusReaderMissing(t *testing.T) {
	reader := FluxStatusReader{Client: fake.NewClientBuilder().Build()}
	_, err := reader.ReadDeploymentStatus(context.Background(), DeploymentStatusRequest{Provider: cicdv1alpha1.GitOpsProviderFlux, ApplicationName: "missing", Namespace: "flux-system"})
	if !errors.Is(err, ErrDeploymentResourceMissing) {
		t.Fatalf("error = %v", err)
	}
}

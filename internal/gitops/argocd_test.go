package gitops

import (
	"context"
	"errors"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestArgoCDStatusReader(t *testing.T) {
	for _, test := range []struct{ name, sync, health string }{
		{name: "synced healthy", sync: "Synced", health: "Healthy"},
		{name: "out of sync", sync: "OutOfSync", health: "Healthy"},
		{name: "degraded", sync: "Synced", health: "Degraded"},
	} {
		t.Run(test.name, func(t *testing.T) {
			application := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "argoproj.io/v1alpha1", "kind": "Application",
				"metadata": map[string]any{"name": "app", "namespace": "argocd"},
				"status":   map[string]any{"sync": map[string]any{"status": test.sync, "revision": "abc123"}, "health": map[string]any{"status": test.health}, "operationState": map[string]any{"phase": "Succeeded"}, "reconciledAt": "2026-08-23T00:00:00Z"},
			}}
			reader := ArgoCDStatusReader{Client: fake.NewClientBuilder().WithRuntimeObjects(application).Build()}
			status, err := reader.ReadDeploymentStatus(context.Background(), DeploymentStatusRequest{Provider: cicdv1alpha1.GitOpsProviderArgoCD, ApplicationName: "app", Namespace: "argocd"})
			if err != nil {
				t.Fatal(err)
			}
			if status.SyncStatus != test.sync || status.HealthStatus != test.health || status.ObservedRevision != "abc123" || status.OperationPhase != "Succeeded" {
				t.Fatalf("status = %#v", status)
			}
		})
	}
}

func TestArgoCDStatusReaderDistinguishesMissingCRDAndApplication(t *testing.T) {
	missing := ArgoCDStatusReader{Client: fake.NewClientBuilder().Build()}
	_, err := missing.ReadDeploymentStatus(context.Background(), DeploymentStatusRequest{Provider: cicdv1alpha1.GitOpsProviderArgoCD, ApplicationName: "app", Namespace: "argocd"})
	if !errors.Is(err, ErrDeploymentResourceMissing) {
		t.Fatalf("error = %v", err)
	}

	noCRD := ArgoCDStatusReader{Client: noMatchClient{Client: fake.NewClientBuilder().Build()}}
	_, err = noCRD.ReadDeploymentStatus(context.Background(), DeploymentStatusRequest{Provider: cicdv1alpha1.GitOpsProviderArgoCD, ApplicationName: "app", Namespace: "argocd"})
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("error = %v", err)
	}
}

type noMatchClient struct{ client.Client }

func (c noMatchClient) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return &meta.NoResourceMatchError{PartialResource: schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}}
}

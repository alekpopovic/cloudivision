package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/rest"
	clientgotesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLocalClusterProvider(t *testing.T) {
	fakeDiscovery := &fakediscovery.FakeDiscovery{Fake: &clientgotesting.Fake{}, FakedServerVersion: &version.Info{GitVersion: "v1.34.2"}}
	result, err := (KubernetesChecker{LocalConfig: &rest.Config{Host: "https://local.example"}, DiscoveryForConfig: func(config *rest.Config) (discovery.DiscoveryInterface, error) {
		if config.Host != "https://local.example" {
			t.Fatalf("host = %q", config.Host)
		}
		return fakeDiscovery, nil
	}}).Check(context.Background(), &cicdv1alpha1.ClusterTarget{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "v1.34.2" {
		t.Fatalf("version = %q", result.Version)
	}
}

func TestMissingKubeconfig(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	checker := KubernetesChecker{Reader: fake.NewClientBuilder().WithScheme(scheme).Build()}
	target := &cicdv1alpha1.ClusterTarget{ObjectMeta: metav1.ObjectMeta{Namespace: "ci"}, Spec: cicdv1alpha1.ClusterTargetSpec{KubeconfigSecretRef: &cicdv1alpha1.SecretKeyRef{Name: "missing"}}}
	if _, err := checker.Check(context.Background(), target); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnreachableCluster(t *testing.T) {
	checker := KubernetesChecker{LocalConfig: &rest.Config{Host: "https://unreachable.example"}, DiscoveryForConfig: func(*rest.Config) (discovery.DiscoveryInterface, error) { return nil, errors.New("connection refused") }}
	if _, err := checker.Check(context.Background(), &cicdv1alpha1.ClusterTarget{}); err == nil {
		t.Fatal("expected unreachable cluster error")
	}
}

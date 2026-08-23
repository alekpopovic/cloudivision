package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestKubernetesProviderResolvesOnlySelectedKey(t *testing.T) {
	provider := testKubernetesProvider(t, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "ci"}, Data: map[string][]byte{"token": []byte("top-secret"), "unrequested": []byte("must-not-leak")}})
	resolved, err := provider.Resolve(context.Background(), SecretRef{Namespace: "ci", Name: "credentials", Keys: []string{"token"}})
	if err != nil {
		t.Fatal(err)
	}
	if string(resolved.Values["token"]) != "top-secret" || resolved.Values["unrequested"] != nil || len(resolved.Keys) != 1 {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestKubernetesProviderErrors(t *testing.T) {
	provider := testKubernetesProvider(t, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "ci"}, Data: map[string][]byte{"token": []byte("top-secret")}})
	for _, test := range []struct {
		name string
		ref  SecretRef
		want error
	}{
		{name: "missing secret", ref: SecretRef{Namespace: "ci", Name: "missing", Keys: []string{"token"}}, want: ErrSecretMissing},
		{name: "missing key", ref: SecretRef{Namespace: "ci", Name: "credentials", Keys: []string{"missing"}}, want: ErrSecretKeyMissing},
		{name: "forbidden namespace", ref: SecretRef{Namespace: "other", Name: "credentials", Keys: []string{"token"}}, want: ErrNamespaceForbidden},
		{name: "all keys implicit", ref: SecretRef{Namespace: "ci", Name: "credentials"}, want: ErrExplicitKeysRequired},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := provider.Resolve(context.Background(), test.ref)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "top-secret") {
				t.Fatalf("secret leaked in error: %v", err)
			}
		})
	}
}

func TestResolvedSecretIsRedactedFromStringAndJSON(t *testing.T) {
	resolved := ResolvedSecret{Name: "credentials", Keys: []string{"token"}, Values: map[string][]byte{"token": []byte("top-secret")}}
	encoded, err := json.Marshal(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(resolved.String(), "top-secret") || strings.Contains(string(encoded), "top-secret") {
		t.Fatalf("secret leaked: string=%q json=%s", resolved.String(), encoded)
	}
}

func TestExternalSecretsHealthDetectsCRD(t *testing.T) {
	crd := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition", "metadata": map[string]any{"name": "externalsecrets.external-secrets.io"}}}
	provider := ExternalSecretsProvider{Client: fake.NewClientBuilder().WithRuntimeObjects(crd).Build()}
	if health := provider.HealthCheck(context.Background()); !health.Healthy {
		t.Fatalf("health = %#v", health)
	}
}

func testKubernetesProvider(t *testing.T, objects ...runtime.Object) KubernetesProvider {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return KubernetesProvider{Client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build(), AllowedNamespaces: []string{"ci"}}
}

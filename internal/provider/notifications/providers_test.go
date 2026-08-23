package notifications

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGenericWebhookPayloadExcludesCredentials(t *testing.T) {
	var payload map[string]any
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type = %q", r.Header.Get("Content-Type"))
		}
		data, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(data, &payload); err != nil {
			t.Error(err)
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	endpoint := "https://notify.example/super-secret-token"
	err := (GenericWebhook{Endpoint: endpoint, Client: client, MaxAttempts: 1}).Send(context.Background(), NotificationRequest{Event: BuildRunSucceeded, Project: "demo", ResourceName: "build-1"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(payload)
	if payload["event"] != string(BuildRunSucceeded) || strings.Contains(string(encoded), "super-secret-token") {
		t.Fatalf("payload = %s", encoded)
	}
}

func TestNotificationEventFiltering(t *testing.T) {
	config := cicdv1alpha1.ProjectNotificationSpec{Events: []string{string(ReleaseFailed)}, Filters: cicdv1alpha1.NotificationFilters{Project: "demo", Environment: "prod", Phase: "FailedProviderStatus"}}
	if !matches(config, NotificationRequest{Event: ReleaseFailed, Project: "demo", Environment: "prod", Phase: "FailedProviderStatus"}) {
		t.Fatal("matching request was filtered")
	}
	if matches(config, NotificationRequest{Event: ReleaseDeployed, Project: "demo", Environment: "prod", Phase: "Deployed"}) {
		t.Fatal("non-matching request passed")
	}
}

func TestDispatcherErrorsRedactWebhookURL(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, fs.ErrInvalid })}
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = cicdv1alpha1.AddToScheme(scheme)
	project := &cicdv1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "ci"}, Spec: cicdv1alpha1.ProjectSpec{Notifications: &cicdv1alpha1.ProjectNotificationSpec{Enabled: true, Provider: "webhook", SecretRef: &cicdv1alpha1.SecretKeyRef{Name: "notify", Key: "url"}}}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "notify", Namespace: "ci"}, Data: map[string][]byte{"url": []byte("https://notify.example/private-token")}}
	dispatcher := KubernetesDispatcher{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(project, secret).Build(), HTTPClient: httpClient}
	err := dispatcher.Notify(context.Background(), NotificationRequest{Event: BuildRunFailed, Project: "demo", Namespace: "ci"})
	if err == nil || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "notify.example") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericWebhookRetriesServerFailures(t *testing.T) {
	attempts := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		attempts++
		status := http.StatusServiceUnavailable
		if attempts < 3 {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}
	err := (GenericWebhook{Endpoint: "https://notify.example", Client: httpClient, MaxAttempts: 3, Backoff: time.Millisecond}).Send(context.Background(), NotificationRequest{Event: BuildRunStarted})
	if err != nil || attempts != 3 {
		t.Fatalf("attempts=%d error=%v", attempts, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

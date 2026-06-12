package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPostBuildRunCreatesCR(t *testing.T) {
	server, k8sClient := newTestServer(t)
	body := `{
		"name":"build-1",
		"namespace":"ci",
		"spec":{
			"projectRef":"project",
			"repositoryRef":"repo",
			"pipelineTemplateRef":"template",
			"revision":"main",
			"triggeredBy":{"type":"api","actor":"test"},
			"image":{"repository":"ghcr.io/cloudivision/app","tag":"main"},
			"executor":"job"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/build-runs", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	created := &cicdv1alpha1.BuildRun{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "build-1", Namespace: "ci"}, created); err != nil {
		t.Fatalf("get created BuildRun: %v", err)
	}
	if created.Spec.RepositoryRef != "repo" {
		t.Fatalf("repositoryRef = %q", created.Spec.RepositoryRef)
	}
}

func TestListBuildRuns(t *testing.T) {
	buildRun := &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "build-1", Namespace: "ci"},
		Spec: cicdv1alpha1.BuildRunSpec{
			ProjectRef:          "project",
			RepositoryRef:       "repo",
			PipelineTemplateRef: "template",
			Revision:            "main",
			TriggeredBy:         cicdv1alpha1.TriggeredBy{Type: cicdv1alpha1.TriggerTypeManual},
			Image:               cicdv1alpha1.ImageRef{Repository: "ghcr.io/cloudivision/app"},
		},
	}
	server, _ := newTestServer(t, buildRun)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs?namespace=ci", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var items []BuildRunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode BuildRuns: %v", err)
	}
	if len(items) != 1 || items[0].Name != "build-1" {
		t.Fatalf("items = %#v, want build-1", items)
	}
}

func TestGitHubWebhookCreatesBuildRun(t *testing.T) {
	body := readFixture(t, "github_push.json")
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "github-event-1")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var buildRuns cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &buildRuns, client.InNamespace("ci")); err != nil {
		t.Fatalf("list BuildRuns: %v", err)
	}
	if len(buildRuns.Items) != 1 {
		t.Fatalf("len(buildRuns.Items) = %d, want 1", len(buildRuns.Items))
	}
	buildRun := buildRuns.Items[0]
	if buildRun.Spec.TriggeredBy.Type != cicdv1alpha1.TriggerTypeWebhook {
		t.Fatalf("trigger type = %q", buildRun.Spec.TriggeredBy.Type)
	}
	if buildRun.Spec.CommitSHA != "1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("commitSHA = %q", buildRun.Spec.CommitSHA)
	}
}

func TestWebhookIdempotencyFallsBackToKubernetesLookup(t *testing.T) {
	body := readFixture(t, "gitlab_push.json")
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitLab)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/gitlab/sample-repository?namespace=ci", bytes.NewReader(body))
		req.Header.Set("X-Gitlab-Token", "webhook-secret")
		req.Header.Set("X-Gitlab-Event-UUID", "gitlab-event-1")
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusCreated {
			t.Fatalf("first status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if i == 1 && rec.Code != http.StatusOK {
			t.Fatalf("second status = %d, body = %s", rec.Code, rec.Body.String())
		}
	}
	var buildRuns cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &buildRuns, client.InNamespace("ci")); err != nil {
		t.Fatalf("list BuildRuns: %v", err)
	}
	if len(buildRuns.Items) != 1 {
		t.Fatalf("len(buildRuns.Items) = %d, want 1", len(buildRuns.Items))
	}
}

func TestGitHubWebhookRejectsInvalidSignature(t *testing.T) {
	body := readFixture(t, "github_push.json")
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "github-event-1")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "wrong-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var buildRuns cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &buildRuns, client.InNamespace("ci")); err != nil {
		t.Fatalf("list BuildRuns: %v", err)
	}
	if len(buildRuns.Items) != 0 {
		t.Fatalf("len(buildRuns.Items) = %d, want 0", len(buildRuns.Items))
	}
}

func TestWebhookRejectsOversizedBody(t *testing.T) {
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	body := strings.Repeat("a", 1<<20+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", strings.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var buildRuns cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &buildRuns, client.InNamespace("ci")); err != nil {
		t.Fatalf("list BuildRuns: %v", err)
	}
	if len(buildRuns.Items) != 0 {
		t.Fatalf("len(buildRuns.Items) = %d, want 0", len(buildRuns.Items))
	}
}

func TestBuildRunLogsReadsPodLogs(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "runner-pod",
			Namespace: "ci",
			Labels: map[string]string{
				"cloudivision.io/buildrun": "build-1",
			},
		},
	}
	server, _ := newTestServer(t, pod)
	server.LogReader = fakeLogReader{data: []byte("one\ntwo\n")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs/ci/build-1/logs?tailLines=2", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var logs LogsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &logs); err != nil {
		t.Fatalf("decode logs response: %v", err)
	}
	if logs.PodName != "runner-pod" || len(logs.Lines) != 2 {
		t.Fatalf("logs response = %#v", logs)
	}
}

func TestBuildRunLogsPodNotFound(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs/ci/missing/logs", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Code != "not_found" {
		t.Fatalf("code = %q, want not_found", errResp.Code)
	}
}

func TestAuditEventsEndpointUsesConfiguredLister(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuditEvents = fakeAuditLister{
		events: []audit.Event{
			{
				ID:        "audit-1",
				Type:      "BuildRunCreated",
				Project:   "project",
				BuildRun:  "build-1",
				Message:   "created",
				CreatedAt: time.Now().UTC(),
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?project=project&buildRun=build-1&type=BuildRunCreated", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var events []AuditEventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil {
		t.Fatalf("decode audit events: %v", err)
	}
	if len(events) != 1 || events[0].ID != "audit-1" {
		t.Fatalf("events = %#v", events)
	}
}

func TestAuthNotDisabledReturnsNotImplemented(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "oidc"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Code != "auth_not_implemented" {
		t.Fatalf("code = %q", errResp.Code)
	}
}

func TestAuthDisabledAllowsRequests(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "disabled"
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCORSAllowsConfiguredLocalUIOrigin(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/build-runs", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:4200" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestErrorResponseIncludesRequestID(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/build-runs", bytes.NewBufferString(`{"namespace":"ci"}`))
	req.Header.Set("X-Request-ID", "req-test")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.RequestID != "req-test" {
		t.Fatalf("requestId = %q, want req-test", errResp.RequestID)
	}
	if got := rec.Header().Get("X-Request-ID"); got != "req-test" {
		t.Fatalf("X-Request-ID = %q, want req-test", got)
	}
}

func TestMetricsEndpointCanBeEnabled(t *testing.T) {
	server, _ := newTestServer(t)
	server.MetricsEnabled = true
	warmup := httptest.NewRecorder()
	server.Handler().ServeHTTP(warmup, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "cloudivision_http_requests_total") {
		t.Fatalf("metrics response did not contain cloudivision HTTP metric")
	}
}

func TestErrorResponsesAreJSON(t *testing.T) {
	server, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/build-runs", bytes.NewBufferString(`{"namespace":"ci"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q", got)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Code == "" || errResp.Message == "" {
		t.Fatalf("error response = %#v", errResp)
	}
}

func newTestServer(t *testing.T, objects ...client.Object) (Server, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add Kubernetes scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
	return Server{
		Client:           k8sClient,
		LogReader:        fakeLogReader{},
		DefaultNamespace: "default",
		AuthMode:         "disabled",
		CORSOrigins:      []string{"http://localhost:4200"},
	}, k8sClient
}

func newWebhookTestServer(t *testing.T, provider cicdv1alpha1.RepositoryProvider) (Server, client.Client) {
	t.Helper()
	project := &cicdv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-project", Namespace: "ci"},
		Spec: cicdv1alpha1.ProjectSpec{
			DisplayName:     "Sample",
			OwnerTeam:       "platform",
			Namespace:       "ci",
			DefaultRegistry: "ghcr.io/cloudivision",
			DefaultBranch:   "main",
			Isolation: cicdv1alpha1.ProjectIsolation{
				CreateNamespace:   false,
				PodSecurityLevel:  cicdv1alpha1.PodSecurityLevelRestricted,
				NetworkPolicyMode: cicdv1alpha1.NetworkPolicyModeDisabled,
			},
		},
	}
	repository := &cicdv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-repository", Namespace: "ci"},
		Spec: cicdv1alpha1.RepositorySpec{
			ProjectRef:          "sample-project",
			Provider:            provider,
			URL:                 "https://github.com/cloudivision/example.git",
			DefaultBranch:       "main",
			PipelineTemplateRef: "sample-template",
			Webhook: cicdv1alpha1.RepositoryWebhook{
				Enabled: true,
				SecretRef: cicdv1alpha1.RequiredSecretKeyRef{
					Name: "webhook-secret",
					Key:  "secret",
				},
			},
		},
	}
	template := &cicdv1alpha1.PipelineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-template", Namespace: "ci"},
		Spec: cicdv1alpha1.PipelineTemplateSpec{
			Build: cicdv1alpha1.PipelineBuildSpec{
				Enabled: true,
				Builder: cicdv1alpha1.BuildBuilderBuildKit,
				Image:   "ghcr.io/cloudivision/example",
				Push:    true,
			},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook-secret", Namespace: "ci"},
		Data: map[string][]byte{
			"secret": []byte("webhook-secret"),
		},
	}
	return newTestServer(t, project, repository, template, secret)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

type fakeLogReader struct {
	data []byte
	err  error
}

func (r fakeLogReader) Logs(context.Context, string, string, *int64) ([]byte, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.data, nil
}

type fakeAuditLister struct {
	events []audit.Event
	filter audit.EventFilter
	err    error
}

func (l fakeAuditLister) ListEvents(_ context.Context, filter audit.EventFilter) ([]audit.Event, error) {
	if l.err != nil {
		return nil, l.err
	}
	l.filter = filter
	return l.events, nil
}

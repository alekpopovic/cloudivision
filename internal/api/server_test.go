package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
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

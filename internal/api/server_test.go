package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/artifacts"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/auth"
	dependencycache "github.com/cloudivision/cloudivision/internal/cache"
	"github.com/cloudivision/cloudivision/internal/logstore"
	"github.com/cloudivision/cloudivision/internal/policy"
	"github.com/cloudivision/cloudivision/internal/provider"
	"github.com/cloudivision/cloudivision/internal/webhook"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestProviderEndpointsExposeCapabilitiesAndHealth(t *testing.T) {
	server, _ := newTestServer(t)
	server.Providers = provider.NewRegistry()
	if err := server.Providers.Register(provider.Static{ProviderName: "generic", ProviderType: "git", Healthy: true, Message: "ready", Features: []provider.Capability{{Name: "clone", Description: "clone repositories"}}}); err != nil {
		t.Fatal(err)
	}

	providersRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(providersRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil))
	if providersRecorder.Code != http.StatusOK {
		t.Fatalf("providers status = %d", providersRecorder.Code)
	}
	var summaries []provider.Summary
	if err := json.Unmarshal(providersRecorder.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].Type != "git" || len(summaries[0].Capabilities) != 1 {
		t.Fatalf("providers = %#v", summaries)
	}

	healthRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(healthRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/providers/health", nil))
	var health []provider.HealthResult
	if err := json.Unmarshal(healthRecorder.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if len(health) != 1 || !health[0].Health.Healthy || health[0].Health.CheckedAt.IsZero() {
		t.Fatalf("health = %#v", health)
	}
}

func TestClusterTargetsIncludeLocalAndExternal(t *testing.T) {
	target := &cicdv1alpha1.ClusterTarget{ObjectMeta: metav1.ObjectMeta{Name: "edge", Namespace: "ci"}, Spec: cicdv1alpha1.ClusterTargetSpec{DisplayName: "Edge", Type: cicdv1alpha1.ClusterTargetTypeDeploy}, Status: cicdv1alpha1.ClusterTargetStatus{Phase: cicdv1alpha1.ClusterTargetPhaseReady, Reachable: true}}
	server, _ := newTestServer(t, target)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/cluster-targets?namespace=ci", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response []ClusterTargetResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response) != 2 || response[0].Name != "local" || response[1].Name != "edge" {
		t.Fatalf("response = %#v", response)
	}
}

func TestOrganizationDirectoryEndpoints(t *testing.T) {
	server, _ := newTestServer(t)
	server.Organizations = auth.NewDevelopmentDirectory()
	for _, path := range []string{"/api/v1/organizations", "/api/v1/organizations/default/teams", "/api/v1/organizations/default/members", "/api/v1/organizations/default/project-access"} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestProjectAPIResponseDoesNotExposeSecretValues(t *testing.T) {
	project := &cicdv1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: "project", Namespace: "ci"}, Spec: cicdv1alpha1.ProjectSpec{DisplayName: "Project", OwnerTeam: "team", Namespace: "ci", DefaultRegistry: "example.com", Isolation: cicdv1alpha1.ProjectIsolation{PodSecurityLevel: cicdv1alpha1.PodSecurityLevelRestricted}, Notifications: &cicdv1alpha1.ProjectNotificationSpec{Enabled: true, Provider: "webhook", SecretRef: &cicdv1alpha1.SecretKeyRef{Name: "notifications", Key: "url"}}}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "notifications", Namespace: "ci"}, Data: map[string][]byte{"url": []byte("https://notify.example/top-secret-token")}}
	server, _ := newTestServer(t, project, secret)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/projects?namespace=ci", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "top-secret-token") || strings.Contains(recorder.Body.String(), "notify.example") {
		t.Fatalf("API exposed secret value: %s", recorder.Body.String())
	}
}

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

func TestCatalogLoadsAndInstallsTemplate(t *testing.T) {
	server, k8sClient := newTestServer(t)
	list := httptest.NewRecorder()
	server.Handler().ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/catalog/pipeline-templates", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"name":"go"`) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	install := httptest.NewRecorder()
	server.Handler().ServeHTTP(install, httptest.NewRequest(http.MethodPost, "/api/v1/catalog/pipeline-templates/go/install", bytes.NewBufferString(`{"namespace":"ci","projectRef":"demo","name":"go-ci"}`)))
	if install.Code != http.StatusCreated {
		t.Fatalf("install status=%d body=%s", install.Code, install.Body.String())
	}
	created := &cicdv1alpha1.PipelineTemplate{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "ci", Name: "go-ci"}, created); err != nil {
		t.Fatal(err)
	}
	if created.Spec.ProjectRef != "demo" || created.Annotations["cicd.cloudivision.io/catalog-version"] == "" {
		t.Fatalf("created = %#v", created)
	}
}

func TestCatalogInstallRejectsInvalidTemplateName(t *testing.T) {
	server, _ := newTestServer(t)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/catalog/pipeline-templates/not-real/install", bytes.NewBufferString(`{}`)))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPipelineTemplateValidationAndUpdate(t *testing.T) {
	template := &cicdv1alpha1.PipelineTemplate{ObjectMeta: metav1.ObjectMeta{Name: "ci", Namespace: "ci"}, Spec: cicdv1alpha1.PipelineTemplateSpec{Steps: []cicdv1alpha1.PipelineStep{{Name: "test", Image: "alpine", Command: []string{"true"}}}, Build: cicdv1alpha1.PipelineBuildSpec{Builder: cicdv1alpha1.BuildBuilderNone}}}
	server, k8sClient := newTestServer(t, template)
	invalid := httptest.NewRecorder()
	server.Handler().ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/v1/pipeline-templates", bytes.NewBufferString(`{"name":"bad","namespace":"ci","spec":{"steps":[{"name":"test","image":"alpine","command":["true"]},{"name":"test","image":"alpine","command":["true"]}],"build":{"enabled":false,"builder":"none","push":false}}}`)))
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "duplicate") {
		t.Fatalf("status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	updated := httptest.NewRecorder()
	server.Handler().ServeHTTP(updated, httptest.NewRequest(http.MethodPut, "/api/v1/pipeline-templates/ci/ci", bytes.NewBufferString(`{"spec":{"description":"updated","steps":[{"name":"lint","image":"alpine","command":["true"]}],"build":{"enabled":false,"builder":"none","push":false},"security":{"runAsNonRoot":true}}}`)))
	if updated.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", updated.Code, updated.Body.String())
	}
	object := &cicdv1alpha1.PipelineTemplate{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "ci", Name: "ci"}, object); err != nil {
		t.Fatal(err)
	}
	if object.Spec.Description != "updated" || object.Spec.Steps[0].Name != "lint" {
		t.Fatalf("object = %#v", object.Spec)
	}
}

func TestPostBuildRunReturnsStructuredPolicyDenial(t *testing.T) {
	server, _ := newTestServer(t)
	server.PolicyEvaluator = denyPolicyEvaluator{}
	body := `{"name":"build-1","namespace":"ci","spec":{"projectRef":"project","repositoryRef":"repo","pipelineTemplateRef":"template","revision":"main","triggeredBy":{"type":"api"},"image":{"repository":"ghcr.io/acme/app"},"executor":"job"}}`
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/build-runs", bytes.NewBufferString(body)))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "policy_denied" || len(response.Violations) != 1 || response.Violations[0].Policy != policy.CanTriggerBuild {
		t.Fatalf("response = %#v", response)
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
	var page PageResponse[BuildRunResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode BuildRuns: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Name != "build-1" || page.Limit != defaultPageLimit {
		t.Fatalf("page = %#v, want build-1 with default limit", page)
	}
}

func TestPurgeProjectCache(t *testing.T) {
	project := &cicdv1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: "project", Namespace: "ci"}}
	server, _ := newTestServer(t, project)
	store := dependencycache.LocalStore{Root: t.TempDir()}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "deps.lock"), []byte("cached"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := dependencycache.Request{Project: "project", Repository: "repo", Key: "key", Paths: []string{"deps.lock"}, Workspace: workspace}
	if err := store.Save(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	server.CacheStore = store
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/projects/project/cache/purge?namespace=ci", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	hit, err := store.Restore(context.Background(), request)
	if err != nil || hit {
		t.Fatalf("Restore() after API purge = %v, %v", hit, err)
	}
}

func TestListBuildRunsPaginatesAndFilters(t *testing.T) {
	objects := make([]client.Object, 0, 125)
	for i := 0; i < 125; i++ {
		phase := cicdv1alpha1.BuildRunPhaseSucceeded
		project := "project-a"
		if i%2 == 0 {
			phase = cicdv1alpha1.BuildRunPhaseFailed
			project = "project-b"
		}
		objects = append(objects, &cicdv1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("build-%03d", i), Namespace: "ci", CreationTimestamp: metav1.NewTime(time.Unix(int64(i), 0))},
			Spec:       cicdv1alpha1.BuildRunSpec{ProjectRef: project, RepositoryRef: "repo", PipelineTemplateRef: "template", Revision: "main", TriggeredBy: cicdv1alpha1.TriggeredBy{Type: cicdv1alpha1.TriggerTypeManual}, Image: cicdv1alpha1.ImageRef{Repository: "example.invalid/app"}},
			Status:     cicdv1alpha1.BuildRunStatus{Phase: phase},
		})
	}
	server, _ := newTestServer(t, objects...)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs?namespace=ci", nil))
	var firstPage PageResponse[BuildRunResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &firstPage); err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Items) != defaultPageLimit || firstPage.TotalCount != 125 || firstPage.NextPageToken == "" {
		t.Fatalf("page = %#v", firstPage)
	}
	if firstPage.Items[0].Name != "build-124" {
		t.Fatalf("first item = %q, want newest build-124", firstPage.Items[0].Name)
	}

	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs?namespace=ci&phase=Succeeded&project=project-a&limit=10", nil))
	var filtered PageResponse[BuildRunResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &filtered); err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 10 || filtered.TotalCount != 62 {
		t.Fatalf("filtered = %#v", filtered)
	}
	for _, item := range filtered.Items {
		if item.Spec.ProjectRef != "project-a" || item.Status.Phase != cicdv1alpha1.BuildRunPhaseSucceeded {
			t.Fatalf("unexpected filtered item %#v", item)
		}
	}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs?namespace=ci&phase=Succeeded&project=project-a&limit=10&pageToken="+filtered.NextPageToken, nil))
	var secondPage PageResponse[BuildRunResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &secondPage); err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Items) != 10 || secondPage.Items[0].Name == filtered.Items[0].Name {
		t.Fatalf("second page = %#v", secondPage)
	}
}

func TestListBuildRunsRejectsInvalidPagination(t *testing.T) {
	server, _ := newTestServer(t)
	for _, query := range []string{"limit=201", "limit=nope", "pageToken=not-base64", "sort=unknown", "order=sideways", "from=bad-time"} {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs?"+query, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("query %q status = %d, body = %s", query, recorder.Code, recorder.Body.String())
		}
	}
}

func TestListReleasesPaginatesAndFilters(t *testing.T) {
	objects := make([]client.Object, 0, 4)
	for i := 0; i < 4; i++ {
		release := testRelease(fmt.Sprintf("release-%d", i))
		release.CreationTimestamp = metav1.NewTime(time.Unix(int64(i), 0))
		release.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
		if i == 0 {
			release.Spec.ProjectRef = "other"
		}
		objects = append(objects, release)
	}
	server, _ := newTestServer(t, objects...)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/releases?namespace=ci&project=project&phase=Deployed&repository=ghcr.io/cloudivision/app&limit=2", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response PageResponse[ReleaseResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 2 || response.TotalCount != 3 || response.NextPageToken == "" {
		t.Fatalf("response = %#v", response)
	}
}

func TestGitHubWebhookCreatesBuildRun(t *testing.T) {
	body := readFixture(t, "github_push.json")
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	recorder := &fakeAuditRecorder{}
	server.Audit = recorder
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
	if !hasAuditType(recorder.events, "WebhookAccepted") || !hasAuditType(recorder.events, "BuildRunCreatedFromWebhook") {
		t.Fatalf("audit events = %#v", recorder.events)
	}
}

func TestGitHubWebhookRejectsMissingSignature(t *testing.T) {
	body := readFixture(t, "github_push.json")
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	recorder := &fakeAuditRecorder{}
	server.Audit = recorder
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "github-event-missing-signature")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	assertBuildRunCount(t, k8sClient, 0)
	if !hasAuditType(recorder.events, "WebhookRejected") {
		t.Fatalf("audit events = %#v", recorder.events)
	}
}

func TestGitHubWebhookDuplicateDeliveryUsesIdempotencyBackend(t *testing.T) {
	body := readFixture(t, "github_push.json")
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	index := newFakeWebhookIndex()
	recorder := &fakeAuditRecorder{}
	server.WebhookIndex = index
	server.Audit = recorder

	for attempt := 0; attempt < 2; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
		req.Header.Set("X-GitHub-Event", "push")
		req.Header.Set("X-GitHub-Delivery", "github-event-duplicate")
		req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, req)
		want := http.StatusCreated
		if attempt == 1 {
			want = http.StatusOK
		}
		if rec.Code != want {
			t.Fatalf("attempt %d status = %d, body = %s", attempt+1, rec.Code, rec.Body.String())
		}
		if attempt == 1 {
			var response WebhookResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Result != "duplicate" || response.Created {
				t.Fatalf("duplicate response = %#v, error = %v", response, err)
			}
		}
	}
	assertBuildRunCount(t, k8sClient, 1)
	if !hasAuditType(recorder.events, "WebhookDuplicate") {
		t.Fatalf("audit events = %#v", recorder.events)
	}
}

func TestGitHubWebhookIgnoresNonDefaultBranch(t *testing.T) {
	body := bytes.ReplaceAll(readFixture(t, "github_push.json"), []byte("refs/heads/main"), []byte("refs/heads/feature"))
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	auditRecorder := &fakeAuditRecorder{}
	server.Audit = auditRecorder
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "github-event-ignored-branch")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response WebhookResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Result != "ignored" || response.Created {
		t.Fatalf("response = %#v, error = %v", response, err)
	}
	assertBuildRunCount(t, k8sClient, 0)
	if !hasAuditReason(auditRecorder.events, "branch_ignored") {
		t.Fatalf("audit events = %#v", auditRecorder.events)
	}
}

func TestGitHubWebhookAcceptsPingWithoutBuildRun(t *testing.T) {
	body := []byte(`{"zen":"Keep it logically awesome.","sender":{"login":"octocat"}}`)
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-GitHub-Delivery", "github-ping-1")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var response WebhookResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || response.Event != "ping" || response.Result != "accepted" || response.BuildRun != nil {
		t.Fatalf("response = %#v, error = %v", response, err)
	}
	assertBuildRunCount(t, k8sClient, 0)
}

func TestGitHubWebhookCreatesBuildRunForPullRequest(t *testing.T) {
	body := []byte(`{"action":"synchronize","pull_request":{"head":{"ref":"feature/payments","sha":"abcdef0123456789"},"base":{"ref":"main"}},"repository":{"clone_url":"https://github.com/cloudivision/example.git"},"sender":{"login":"octocat"}}`)
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", "github-pr-1")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var runs cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &runs, client.InNamespace("ci")); err != nil {
		t.Fatal(err)
	}
	if len(runs.Items) != 1 || runs.Items[0].Spec.Branch != "feature/payments" || runs.Items[0].Spec.CommitSHA != "abcdef0123456789" {
		t.Fatalf("BuildRuns = %#v", runs.Items)
	}
}

func TestGitHubWebhookRejectsMalformedPayload(t *testing.T) {
	body := []byte(`{"ref":`)
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "github-malformed-1")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	assertBuildRunCount(t, k8sClient, 0)
}

func TestGitHubWebhookRejectsStaleTimestamp(t *testing.T) {
	pushedAt := time.Now().UTC().Add(-25 * time.Hour).Unix()
	body := []byte(fmt.Sprintf(`{"ref":"refs/heads/main","after":"abcdef0123456789","repository":{"pushed_at":%d},"sender":{"login":"octocat"}}`, pushedAt))
	server, k8sClient := newWebhookTestServer(t, cicdv1alpha1.RepositoryProviderGitHub)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/github/sample-repository?namespace=ci", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "github-stale-1")
	req.Header.Set("X-Hub-Signature-256", webhook.SignGitHub(body, "webhook-secret"))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	assertBuildRunCount(t, k8sClient, 0)
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

	if rec.Code != http.StatusRequestEntityTooLarge {
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

func TestBuildRunLogsReadsConfiguredStoreBeforePods(t *testing.T) {
	store := logstore.NewMemoryStore()
	if err := store.Append(context.Background(), logstore.AppendLogRequest{Namespace: "ci", BuildRun: "build-1", Lines: []logstore.LogLine{{Timestamp: time.Unix(10, 0), Step: "test", Message: "stored"}}}); err != nil {
		t.Fatal(err)
	}
	server, _ := newTestServer(t)
	server.LogStore = store
	server.LogReader = fakeLogReader{err: errors.New("pod logs must not be read")}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs/ci/build-1/logs?tailLines=1&step=test", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var response LogsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Backend != "memory" || len(response.Lines) != 1 || !strings.Contains(response.Lines[0], "[step:test] stored") {
		t.Fatalf("response = %#v", response)
	}
}

func TestBuildRunArtifactsListsStatusAndDownloadsContent(t *testing.T) {
	store := artifacts.NewMemoryStore()
	ref, err := store.Put(context.Background(), artifacts.PutArtifactRequest{Namespace: "ci", BuildRun: "build-1", Name: "report.xml", Path: "reports/report.xml", Type: "application/xml", Digest: "sha256:abc", Data: []byte("<testsuite/>")})
	if err != nil {
		t.Fatal(err)
	}
	createdAt := metav1.NewTime(ref.CreatedAt)
	buildRun := testActionBuildRun("build-1", cicdv1alpha1.BuildRunPhaseSucceeded)
	buildRun.Status.Artifacts = []cicdv1alpha1.BuildRunArtifactStatus{{Name: ref.Name, Path: ref.Path, Type: ref.Type, Size: ref.Size, Digest: ref.Digest, Ref: ref.Ref, CreatedAt: &createdAt}}
	server, _ := newTestServer(t, buildRun)
	server.ArtifactStore = store

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs/ci/build-1/artifacts", nil))
	if listRecorder.Code != http.StatusOK || !strings.Contains(listRecorder.Body.String(), "report.xml") {
		t.Fatalf("list status=%d body=%s", listRecorder.Code, listRecorder.Body.String())
	}

	downloadRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(downloadRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/build-runs/ci/build-1/artifacts/report.xml", nil))
	if downloadRecorder.Code != http.StatusOK || downloadRecorder.Body.String() != "<testsuite/>" || downloadRecorder.Header().Get("Content-Type") != "application/xml" {
		t.Fatalf("download status=%d headers=%v body=%s", downloadRecorder.Code, downloadRecorder.Header(), downloadRecorder.Body.String())
	}
}

func TestCancelRunningBuildRunIsIdempotent(t *testing.T) {
	buildRun := testActionBuildRun("build-running", cicdv1alpha1.BuildRunPhaseRunning)
	buildRun.Status.JobRef = cicdv1alpha1.ObjectRef{Name: "build-running-job", Namespace: "ci"}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: "build-running-job", Namespace: "ci"}}
	server, k8sClient := newTestServer(t, buildRun, job)
	for attempt := 0; attempt < 2; attempt++ {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/build-runs/ci/build-running/cancel", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d body = %s", attempt+1, recorder.Code, recorder.Body.String())
		}
	}
	var updated cicdv1alpha1.BuildRun
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: buildRun.Name, Namespace: buildRun.Namespace}, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseCancelled || updated.Status.CompletedAt == nil {
		t.Fatalf("status = %#v", updated.Status)
	}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: job.Name, Namespace: job.Namespace}, &batchv1.Job{}); !apierrors.IsNotFound(err) {
		t.Fatalf("Job get error = %v, want NotFound", err)
	}
}

func TestRetryFailedBuildRunCreatesRelatedRun(t *testing.T) {
	server, k8sClient := newTestServer(t, testActionBuildRun("build-failed", cicdv1alpha1.BuildRunPhaseFailed))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/build-runs/ci/build-failed/retry", nil))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	assertRelatedBuildRun(t, k8sClient, retryOfAnnotation, "build-failed")
}

func TestRerunSucceededBuildRunCreatesRelatedRun(t *testing.T) {
	server, k8sClient := newTestServer(t, testActionBuildRun("build-succeeded", cicdv1alpha1.BuildRunPhaseSucceeded))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/build-runs/ci/build-succeeded/rerun", nil))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	assertRelatedBuildRun(t, k8sClient, rerunOfAnnotation, "build-succeeded")
}

func TestRetryRunningBuildRunIsDenied(t *testing.T) {
	server, k8sClient := newTestServer(t, testActionBuildRun("build-running", cicdv1alpha1.BuildRunPhaseRunning))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/build-runs/ci/build-running/retry", nil))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	assertBuildRunCount(t, k8sClient, 1)
}

func testActionBuildRun(name string, phase cicdv1alpha1.BuildRunPhase) *cicdv1alpha1.BuildRun {
	return &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ci"},
		Spec:       cicdv1alpha1.BuildRunSpec{ProjectRef: "project", RepositoryRef: "repository", PipelineTemplateRef: "pipeline", Revision: "main", TriggeredBy: cicdv1alpha1.TriggeredBy{Type: cicdv1alpha1.TriggerTypeManual}, Image: cicdv1alpha1.ImageRef{Repository: "example.invalid/app"}},
		Status:     cicdv1alpha1.BuildRunStatus{Phase: phase},
	}
}

func assertRelatedBuildRun(t *testing.T, k8sClient client.Client, annotation, source string) {
	t.Helper()
	var list cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &list, client.InNamespace("ci")); err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("BuildRun count = %d, want 2", len(list.Items))
	}
	for _, item := range list.Items {
		if item.Name != source && item.Annotations[annotation] == source {
			return
		}
	}
	t.Fatalf("related BuildRun with %s=%s not found", annotation, source)
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
	var page PageResponse[AuditEventResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode audit events: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "audit-1" {
		t.Fatalf("events = %#v", page.Items)
	}
}

func TestAuditExportJSONAndCSVFiltersAndRedactsSecrets(t *testing.T) {
	now := time.Now().UTC()
	server, _ := newTestServer(t)
	server.AuditEvents = fakeAuditLister{events: []audit.Event{
		{ID: "one", Type: "WebhookRejected", Actor: "alice", Organization: "acme", Project: "store", Message: "token=super-secret", Metadata: json.RawMessage(`{"token":"super-secret","nested":{"password":"hidden"}}`), CreatedAt: now},
		{ID: "two", Type: "BuildRunCreated", Actor: "bob", Organization: "other", Project: "other", CreatedAt: now},
	}}
	for _, format := range []string{"json", "csv"} {
		recorder := httptest.NewRecorder()
		path := "/api/v1/audit/events/export?format=" + format + "&organization=acme&project=store&actor=alice&type=WebhookRejected&from=" + now.Add(-time.Minute).Format(time.RFC3339)
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", format, recorder.Code, recorder.Body.String())
		}
		body := recorder.Body.String()
		if !strings.Contains(body, "one") || strings.Contains(body, "two") || strings.Contains(body, "super-secret") || strings.Contains(body, "hidden") {
			t.Fatalf("unsafe or unfiltered %s export: %s", format, body)
		}
		if format == "csv" && !strings.Contains(recorder.Header().Get("Content-Type"), "text/csv") {
			t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
		}
	}
}

func TestAuditExportPermissionChecks(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "oidc"
	server.Authenticator = fakeAuthenticator{principal: &auth.Principal{Subject: "viewer", Roles: []auth.Role{auth.RoleViewer}}}
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/export", nil))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("viewer status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	server.Authenticator = fakeAuthenticator{principal: &auth.Principal{Subject: "auditor", Roles: []auth.Role{auth.RoleAuditor}}}
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/export", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("auditor status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestComplianceReports(t *testing.T) {
	started := metav1.NewTime(time.Now().UTC().Add(-2 * time.Minute))
	completed := metav1.NewTime(started.Add(time.Minute))
	succeeded := &cicdv1alpha1.BuildRun{ObjectMeta: metav1.ObjectMeta{Name: "success", Namespace: "ci", CreationTimestamp: started}, Spec: cicdv1alpha1.BuildRunSpec{ProjectRef: "store", RepositoryRef: "repo"}, Status: cicdv1alpha1.BuildRunStatus{Phase: cicdv1alpha1.BuildRunPhaseSucceeded, StartedAt: &started, CompletedAt: &completed}}
	failed := &cicdv1alpha1.BuildRun{ObjectMeta: metav1.ObjectMeta{Name: "failed", Namespace: "ci", CreationTimestamp: started}, Spec: cicdv1alpha1.BuildRunSpec{ProjectRef: "store", RepositoryRef: "repo"}, Status: cicdv1alpha1.BuildRunStatus{Phase: cicdv1alpha1.BuildRunPhaseFailed, Failure: cicdv1alpha1.FailureStatus{Reason: "TestsFailed"}}}
	release := &cicdv1alpha1.Release{ObjectMeta: metav1.ObjectMeta{Name: "release", Namespace: "ci", CreationTimestamp: started}, Spec: cicdv1alpha1.ReleaseSpec{ProjectRef: "store", EnvironmentRef: "prod", RollbackOf: "old"}, Status: cicdv1alpha1.ReleaseStatus{Phase: cicdv1alpha1.ReleasePhaseFailedProviderStatus}}
	server, _ := newTestServer(t, succeeded, failed, release)
	server.AuditEvents = fakeAuditLister{events: []audit.Event{{Type: "ReleaseApproved", Project: "store", CreatedAt: time.Now().UTC()}, {Type: "WebhookRejected", Project: "store", CreatedAt: time.Now().UTC()}, {Type: "PolicyDenied", Project: "store", Message: "unsigned release blocked", CreatedAt: time.Now().UTC()}, {Type: "CriticalVulnerabilityBlocked", Project: "store", Message: "critical vulnerability blocked", CreatedAt: time.Now().UTC()}}}
	buildRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(buildRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/reports/builds?namespace=ci&project=store", nil))
	var buildReport BuildReport
	if err := json.Unmarshal(buildRecorder.Body.Bytes(), &buildReport); err != nil {
		t.Fatal(err)
	}
	if buildReport.Total != 2 || buildReport.Succeeded != 1 || buildReport.FailedByReason["TestsFailed"] != 1 || buildReport.AverageDurationSeconds != 60 {
		t.Fatalf("build report = %#v", buildReport)
	}
	releaseRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(releaseRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/reports/releases?namespace=ci&project=store", nil))
	var releaseReport ReleaseReport
	if err := json.Unmarshal(releaseRecorder.Body.Bytes(), &releaseReport); err != nil {
		t.Fatal(err)
	}
	if releaseReport.Total != 1 || releaseReport.Approvals != 1 || releaseReport.Rollbacks != 1 || releaseReport.DeploymentFailures != 1 {
		t.Fatalf("release report = %#v", releaseReport)
	}
	securityRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(securityRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/reports/security?project=store", nil))
	var securityReport SecurityReport
	if err := json.Unmarshal(securityRecorder.Body.Bytes(), &securityReport); err != nil {
		t.Fatal(err)
	}
	if securityReport.PolicyDenials != 1 || securityReport.WebhookRejections != 1 || securityReport.UnsignedReleasesBlocked != 1 || securityReport.CriticalVulnerabilityBlocks != 1 {
		t.Fatalf("security report = %#v", securityReport)
	}
}

func TestApproveReleasePatchesSpecAndRecordsAudit(t *testing.T) {
	environment := &cicdv1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "ci"},
		Spec: cicdv1alpha1.EnvironmentSpec{
			ProjectRef:       "project",
			DisplayName:      "Production",
			Namespace:        "prod",
			Type:             cicdv1alpha1.EnvironmentTypeProduction,
			RequiresApproval: true,
			GitOps:           cicdv1alpha1.EnvironmentGitOpsSpec{Provider: cicdv1alpha1.GitOpsProviderGeneric},
		},
	}
	release := testRelease("release-1")
	recorder := &fakeAuditRecorder{}
	server, k8sClient := newTestServer(t, environment, release)
	server.Audit = recorder
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/release-1/approve", bytes.NewBufferString(`{"actor":"alice","comment":"ship it"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated := &cicdv1alpha1.Release{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "release-1", Namespace: "ci"}, updated); err != nil {
		t.Fatalf("get Release: %v", err)
	}
	if !updated.Spec.Approval.Required || updated.Spec.Approval.ApprovedBy != "dev-user@localhost" || updated.Spec.Approval.ApprovedAt == nil {
		t.Fatalf("approval = %#v, want approved by dev-user@localhost", updated.Spec.Approval)
	}
	if updated.Annotations["cloudivision.io/approval-action"] != "approved" {
		t.Fatalf("annotations = %#v, want approved action", updated.Annotations)
	}
	if len(recorder.events) != 1 || recorder.events[0].Type != "ReleaseApproved" || recorder.events[0].Actor != "dev-user@localhost" {
		t.Fatalf("audit events = %#v", recorder.events)
	}
}

func TestRejectReleaseMarksFailedAndRecordsAudit(t *testing.T) {
	release := testRelease("release-1")
	release.Spec.Approval.Required = true
	recorder := &fakeAuditRecorder{}
	server, k8sClient := newTestServer(t, release)
	server.Audit = recorder
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/release-1/reject", bytes.NewBufferString(`{"actor":"bob","comment":"hold"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated := &cicdv1alpha1.Release{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "release-1", Namespace: "ci"}, updated); err != nil {
		t.Fatalf("get Release: %v", err)
	}
	if updated.Spec.Approval.RejectedBy != "dev-user@localhost" || updated.Spec.Approval.RejectedAt == nil {
		t.Fatalf("approval = %#v, want rejected by dev-user@localhost", updated.Spec.Approval)
	}
	if updated.Status.Phase != cicdv1alpha1.ReleasePhaseFailedApproval {
		t.Fatalf("phase = %q, want FailedApproval", updated.Status.Phase)
	}
	if len(recorder.events) != 1 || recorder.events[0].Type != "ReleaseRejected" || recorder.events[0].Actor != "dev-user@localhost" {
		t.Fatalf("audit events = %#v", recorder.events)
	}
}

func TestPromoteReleaseCreatesStagingRelease(t *testing.T) {
	source := testRelease("dev-release")
	source.Spec.EnvironmentRef = "dev"
	source.Spec.Image.Digest = "sha256:abc"
	source.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	target := &cicdv1alpha1.Environment{ObjectMeta: metav1.ObjectMeta{Name: "staging", Namespace: "ci"}, Spec: cicdv1alpha1.EnvironmentSpec{ProjectRef: "project", DisplayName: "Staging", Namespace: "staging", Type: cicdv1alpha1.EnvironmentTypeStaging, GitOps: cicdv1alpha1.EnvironmentGitOpsSpec{Provider: cicdv1alpha1.GitOpsProviderGeneric}}}
	recorder := &fakeAuditRecorder{}
	server, k8sClient := newTestServer(t, source, target)
	server.Audit = recorder
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/dev-release/promote", bytes.NewBufferString(`{"targetEnvironmentRef":"staging","actor":"alice"}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response ReleaseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	created := &cicdv1alpha1.Release{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "ci", Name: response.Name}, created); err != nil {
		t.Fatal(err)
	}
	if created.Spec.EnvironmentRef != "staging" || created.Spec.Image.Digest != "sha256:abc" || created.Spec.PromotedFrom != source.Name {
		t.Fatalf("created = %#v", created.Spec)
	}
	if !hasAuditType(recorder.events, "ReleasePromoted") {
		t.Fatalf("events = %#v", recorder.events)
	}
}

func TestPromoteReleaseToProductionAwaitsApproval(t *testing.T) {
	source := testRelease("staging-release")
	source.Spec.Image.Digest = "sha256:abc"
	source.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	target := &cicdv1alpha1.Environment{ObjectMeta: metav1.ObjectMeta{Name: "production", Namespace: "ci"}, Spec: cicdv1alpha1.EnvironmentSpec{ProjectRef: "project", DisplayName: "Production", Namespace: "production", Type: cicdv1alpha1.EnvironmentTypeProduction, RequiresApproval: true, GitOps: cicdv1alpha1.EnvironmentGitOpsSpec{Provider: cicdv1alpha1.GitOpsProviderGeneric}}}
	server, _ := newTestServer(t, source, target)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/staging-release/promote", bytes.NewBufferString(`{"targetEnvironmentRef":"production"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response ReleaseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Spec.Approval.Required || response.Status.Phase != "" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRollbackCreatesImmutableReplacement(t *testing.T) {
	failed := testRelease("failed-release")
	failed.Spec.Image.Digest = "sha256:new"
	failed.Status.Phase = cicdv1alpha1.ReleasePhaseFailedProviderStatus
	previous := testRelease("previous-release")
	previous.Spec.Image.Digest = "sha256:old"
	previous.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	environment := &cicdv1alpha1.Environment{ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "ci"}, Spec: cicdv1alpha1.EnvironmentSpec{ProjectRef: "project", DisplayName: "Prod", Namespace: "prod", Type: cicdv1alpha1.EnvironmentTypeProduction, RequiresApproval: true, GitOps: cicdv1alpha1.EnvironmentGitOpsSpec{Provider: cicdv1alpha1.GitOpsProviderGeneric}, Policy: cicdv1alpha1.EnvironmentPolicySpec{RequireImageDigest: true}}}
	server, k8sClient := newTestServer(t, failed, previous, environment)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/failed-release/rollback", bytes.NewBufferString(`{"targetReleaseRef":"previous-release","reason":"health regression"}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var response ReleaseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Spec.RollbackOf != failed.Name || response.Spec.RollbackTo != previous.Name || response.Spec.Image.Digest != "sha256:old" {
		t.Fatalf("response = %#v", response.Spec)
	}
	unchanged := &cicdv1alpha1.Release{}
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "ci", Name: failed.Name}, unchanged); err != nil {
		t.Fatal(err)
	}
	if unchanged.Spec.Image.Digest != "sha256:new" {
		t.Fatalf("source mutated = %#v", unchanged.Spec)
	}
}

func TestRollbackBlockedByDigestPolicy(t *testing.T) {
	failed := testRelease("failed-release")
	previous := testRelease("previous-release")
	previous.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	environment := &cicdv1alpha1.Environment{ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "ci"}, Spec: cicdv1alpha1.EnvironmentSpec{ProjectRef: "project", DisplayName: "Prod", Namespace: "prod", Type: cicdv1alpha1.EnvironmentTypeProduction, RequiresApproval: true, GitOps: cicdv1alpha1.EnvironmentGitOpsSpec{Provider: cicdv1alpha1.GitOpsProviderGeneric}, Policy: cicdv1alpha1.EnvironmentPolicySpec{RequireImageDigest: true}}}
	server, _ := newTestServer(t, failed, previous, environment)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/failed-release/rollback", bytes.NewBufferString(`{"targetReleaseRef":"previous-release"}`)))
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "no digest") {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestApproveReleaseRejectsDeployedRelease(t *testing.T) {
	release := testRelease("release-1")
	release.Spec.Approval.Required = true
	release.Status.Phase = cicdv1alpha1.ReleasePhaseDeployed
	server, _ := newTestServer(t, release)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/releases/ci/release-1/approve", bytes.NewBufferString(`{"actor":"alice"}`))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOIDCModeRejectsMissingBearerToken(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "oidc"
	server.Authenticator = fakeAuthenticator{err: auth.ErrMissingBearerToken}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Code != "unauthorized" {
		t.Fatalf("code = %q", errResp.Code)
	}
}

func TestOIDCModeAllowsValidViewerToRead(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "oidc"
	server.Authenticator = fakeAuthenticator{principal: &auth.Principal{Subject: "user-1", Roles: []auth.Role{auth.RoleViewer}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestViewerCannotCreateBuildRun(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "oidc"
	server.Authenticator = fakeAuthenticator{principal: &auth.Principal{Subject: "viewer", Roles: []auth.Role{auth.RoleViewer}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/build-runs", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestAuthDisabledAllowsRequestsAndMarksDevelopmentMode(t *testing.T) {
	server, _ := newTestServer(t)
	server.AuthMode = "disabled"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/build-runs", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("X-Cloudivision-Auth-Mode"); got != "development" {
		t.Fatalf("X-Cloudivision-Auth-Mode = %q, want development", got)
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
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}).
		WithStatusSubresource(&cicdv1alpha1.Release{}).
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
				Enabled:     true,
				PullRequest: cicdv1alpha1.RepositoryPullRequestFilters{Enabled: true},
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

func testRelease(name string) *cicdv1alpha1.Release {
	return &cicdv1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ci"},
		Spec: cicdv1alpha1.ReleaseSpec{
			ProjectRef:     "project",
			EnvironmentRef: "prod",
			BuildRunRef:    "build-1",
			Image:          cicdv1alpha1.ImageRef{Repository: "ghcr.io/cloudivision/app", Tag: "main"},
			Strategy:       cicdv1alpha1.ReleaseStrategyGitOps,
		},
	}
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

type fakeAuditRecorder struct {
	events []audit.Event
}

func (r *fakeAuditRecorder) Record(_ context.Context, event audit.Event) error {
	r.events = append(r.events, event)
	return nil
}

func hasAuditType(events []audit.Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func hasAuditReason(events []audit.Event, reason string) bool {
	for _, event := range events {
		var metadata map[string]string
		if json.Unmarshal(event.Metadata, &metadata) == nil && metadata["reason"] == reason {
			return true
		}
	}
	return false
}

func assertBuildRunCount(t *testing.T, k8sClient client.Client, want int) {
	t.Helper()
	var buildRuns cicdv1alpha1.BuildRunList
	if err := k8sClient.List(context.Background(), &buildRuns, client.InNamespace("ci")); err != nil {
		t.Fatalf("list BuildRuns: %v", err)
	}
	if len(buildRuns.Items) != want {
		t.Fatalf("len(BuildRuns) = %d, want %d", len(buildRuns.Items), want)
	}
}

type fakeWebhookIndex struct {
	events map[string]audit.WebhookEvent
}

func newFakeWebhookIndex() *fakeWebhookIndex {
	return &fakeWebhookIndex{events: map[string]audit.WebhookEvent{}}
}

func (i *fakeWebhookIndex) FindWebhookEvent(_ context.Context, provider, repository, eventID string) (*audit.WebhookEvent, error) {
	event, ok := i.events[provider+"/"+repository+"/"+eventID]
	if !ok {
		return nil, nil
	}
	return &event, nil
}

func (i *fakeWebhookIndex) RecordWebhookEvent(_ context.Context, event audit.WebhookEvent) error {
	i.events[event.Provider+"/"+event.Repository+"/"+event.EventID] = event
	return nil
}

type fakeAuthenticator struct {
	principal *auth.Principal
	err       error
}

type denyPolicyEvaluator struct{}

func (denyPolicyEvaluator) EvaluateBuildRun(context.Context, policy.BuildRunPolicyInput) policy.Decision {
	return policy.Decision{Allowed: false, Reason: "PolicyDenied", Message: "build trigger denied", Violations: []policy.Violation{{Policy: policy.CanTriggerBuild, Severity: "error", Message: "build trigger denied", FieldPath: "spec.triggeredBy"}}}
}

func (denyPolicyEvaluator) EvaluateRelease(context.Context, policy.ReleasePolicyInput) policy.Decision {
	return policy.Decision{Allowed: true}
}

func (denyPolicyEvaluator) EvaluatePipelineTemplate(context.Context, policy.PipelineTemplatePolicyInput) policy.Decision {
	return policy.Decision{Allowed: true}
}

func (a fakeAuthenticator) Authenticate(*http.Request) (*auth.Principal, error) {
	if a.err != nil {
		return nil, a.err
	}
	return a.principal, nil
}

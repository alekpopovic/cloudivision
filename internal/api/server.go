package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/auth"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/observability"
	"github.com/cloudivision/cloudivision/internal/redact"
	"github.com/cloudivision/cloudivision/internal/webhook"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Server struct {
	Client           client.Client
	LogReader        PodLogReader
	Logger           *slog.Logger
	Audit            audit.Recorder
	AuditEvents      audit.EventLister
	WebhookIndex     audit.WebhookIndexer
	Authenticator    auth.Authenticator
	DefaultNamespace string
	AuthMode         string
	CORSOrigins      []string
	MetricsEnabled   bool
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.health)
	if s.MetricsEnabled {
		mux.Handle("GET /metrics", observability.MetricsHandler())
	}
	mux.HandleFunc("GET /api/v1/auth/me", s.currentUser)
	mux.HandleFunc("GET /api/v1/projects", s.projects)
	mux.HandleFunc("POST /api/v1/projects", s.projects)
	mux.HandleFunc("GET /api/v1/projects/{name}", s.project)
	mux.HandleFunc("GET /api/v1/repositories", s.repositories)
	mux.HandleFunc("POST /api/v1/repositories", s.repositories)
	mux.HandleFunc("GET /api/v1/pipeline-templates", s.pipelineTemplates)
	mux.HandleFunc("POST /api/v1/pipeline-templates", s.pipelineTemplates)
	mux.HandleFunc("GET /api/v1/build-runs", s.buildRuns)
	mux.HandleFunc("POST /api/v1/build-runs", s.buildRuns)
	mux.HandleFunc("GET /api/v1/build-runs/{namespace}/{name}", s.buildRun)
	mux.HandleFunc("GET /api/v1/build-runs/{namespace}/{name}/logs", s.buildRunLogs)
	mux.HandleFunc("GET /api/v1/environments", s.environments)
	mux.HandleFunc("GET /api/v1/releases", s.releases)
	mux.HandleFunc("POST /api/v1/releases/{namespace}/{name}/approve", s.approveRelease)
	mux.HandleFunc("POST /api/v1/releases/{namespace}/{name}/reject", s.rejectRelease)
	mux.HandleFunc("GET /api/v1/audit/events", s.auditEvents)
	mux.HandleFunc("POST /api/v1/webhooks/github/{repositoryName}", s.webhook(webhook.ProviderGitHub))
	mux.HandleFunc("POST /api/v1/webhooks/gitlab/{repositoryName}", s.webhook(webhook.ProviderGitLab))
	mux.HandleFunc("POST /api/v1/webhooks/gitea/{repositoryName}", s.webhook(webhook.ProviderGitea))
	mux.HandleFunc("POST /api/v1/webhooks/generic/{repositoryName}", s.webhook(webhook.ProviderGeneric))
	return s.requestID(s.logging(s.cors(s.auth(mux))))
}

func (s Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func (s Server) currentUser(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFromContext(r.Context())
	writeJSON(w, http.StatusOK, principalDTO(principal))
}

func (s Server) projects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var list cicdv1alpha1.ProjectList
		if err := s.list(r.Context(), r, &list); err != nil {
			s.writeError(w, err)
			return
		}
		items := make([]ProjectResponse, 0, len(list.Items))
		for _, item := range list.Items {
			items = append(items, projectDTO(item))
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req ProjectRequest
		if !s.decode(w, r, &req) {
			return
		}
		if err := validateName(req.Name); err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		if req.Spec.DisplayName == "" || req.Spec.OwnerTeam == "" || req.Spec.Namespace == "" || req.Spec.DefaultRegistry == "" {
			s.writeError(w, badRequest("displayName, ownerTeam, namespace and defaultRegistry are required"))
			return
		}
		obj := &cicdv1alpha1.Project{ObjectMeta: objectMeta(req.Name, s.namespace(req.Namespace)), Spec: req.Spec}
		if err := s.Client.Create(r.Context(), obj); err != nil {
			s.writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, projectDTO(*obj))
	}
}

func (s Server) project(w http.ResponseWriter, r *http.Request) {
	var obj cicdv1alpha1.Project
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: s.namespace(r.URL.Query().Get("namespace"))}, &obj); err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectDTO(obj))
}

func (s Server) repositories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var list cicdv1alpha1.RepositoryList
		if err := s.list(r.Context(), r, &list); err != nil {
			s.writeError(w, err)
			return
		}
		items := make([]RepositoryResponse, 0, len(list.Items))
		for _, item := range list.Items {
			items = append(items, repositoryDTO(item))
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req RepositoryRequest
		if !s.decode(w, r, &req) {
			return
		}
		if err := validateName(req.Name); err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		if req.Spec.ProjectRef == "" || req.Spec.URL == "" || req.Spec.PipelineTemplateRef == "" {
			s.writeError(w, badRequest("projectRef, url and pipelineTemplateRef are required"))
			return
		}
		obj := &cicdv1alpha1.Repository{ObjectMeta: objectMeta(req.Name, s.namespace(req.Namespace)), Spec: req.Spec}
		if err := s.Client.Create(r.Context(), obj); err != nil {
			s.writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, repositoryDTO(*obj))
	}
}

func (s Server) pipelineTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var list cicdv1alpha1.PipelineTemplateList
		if err := s.list(r.Context(), r, &list); err != nil {
			s.writeError(w, err)
			return
		}
		items := make([]PipelineTemplateResponse, 0, len(list.Items))
		for _, item := range list.Items {
			items = append(items, pipelineTemplateDTO(item))
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req PipelineTemplateRequest
		if !s.decode(w, r, &req) {
			return
		}
		if err := validateName(req.Name); err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		if req.Spec.Build.Enabled && req.Spec.Build.Builder == "" {
			s.writeError(w, badRequest("build.builder is required when build.enabled is true"))
			return
		}
		obj := &cicdv1alpha1.PipelineTemplate{ObjectMeta: objectMeta(req.Name, s.namespace(req.Namespace)), Spec: req.Spec}
		if err := s.Client.Create(r.Context(), obj); err != nil {
			s.writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, pipelineTemplateDTO(*obj))
	}
}

func (s Server) buildRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var list cicdv1alpha1.BuildRunList
		if err := s.list(r.Context(), r, &list); err != nil {
			s.writeError(w, err)
			return
		}
		items := make([]BuildRunResponse, 0, len(list.Items))
		for _, item := range list.Items {
			items = append(items, buildRunDTO(item))
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var req BuildRunRequest
		if !s.decode(w, r, &req) {
			return
		}
		if err := validateBuildRunRequest(req); err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		obj := &cicdv1alpha1.BuildRun{ObjectMeta: objectMeta(req.Name, s.namespace(req.Namespace)), Spec: req.Spec}
		if err := s.Client.Create(r.Context(), obj); err != nil {
			s.writeError(w, err)
			return
		}
		s.recordAudit(r.Context(), audit.Event{
			Type:     "BuildRunCreated",
			Actor:    auth.ActorFromContext(r.Context(), obj.Spec.TriggeredBy.Actor),
			Project:  obj.Spec.ProjectRef,
			BuildRun: obj.Name,
			Message:  "Created BuildRun through API.",
		})
		writeJSON(w, http.StatusCreated, buildRunDTO(*obj))
	}
}

func (s Server) buildRun(w http.ResponseWriter, r *http.Request) {
	var obj cicdv1alpha1.BuildRun
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: r.PathValue("namespace")}, &obj); err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildRunDTO(obj))
}

func (s Server) buildRunLogs(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")
	tailLines, err := parseTailLines(r.URL.Query().Get("tailLines"))
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	var pods corev1.PodList
	selector := labels.SelectorFromSet(labels.Set{"cloudivision.io/buildrun": name})
	if err := s.Client.List(r.Context(), &pods, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		s.writeError(w, err)
		return
	}
	if len(pods.Items) == 0 {
		s.writeError(w, notFound("runner pod not found"))
		return
	}
	if s.LogReader == nil {
		s.writeError(w, internalError("pod log reader is not configured"))
		return
	}
	pod := pods.Items[0]
	data, err := s.LogReader.Logs(r.Context(), namespace, pod.Name, tailLines)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, LogsResponse{
		Namespace: namespace,
		BuildRun:  name,
		PodName:   pod.Name,
		Lines:     splitLogLines(string(data)),
	})
}

func (s Server) environments(w http.ResponseWriter, r *http.Request) {
	var list cicdv1alpha1.EnvironmentList
	if err := s.list(r.Context(), r, &list); err != nil {
		s.writeError(w, err)
		return
	}
	items := make([]EnvironmentResponse, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, environmentDTO(item))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) releases(w http.ResponseWriter, r *http.Request) {
	var list cicdv1alpha1.ReleaseList
	if err := s.list(r.Context(), r, &list); err != nil {
		s.writeError(w, err)
		return
	}
	items := make([]ReleaseResponse, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, releaseDTO(item))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) approveRelease(w http.ResponseWriter, r *http.Request) {
	release, req, ok := s.releaseApprovalActionInput(w, r)
	if !ok {
		return
	}
	if release.Status.Phase == cicdv1alpha1.ReleasePhaseDeployed || release.Status.Phase == cicdv1alpha1.ReleasePhaseDeploying {
		s.writeError(w, conflict("release is already deploying or deployed"))
		return
	}
	if release.Spec.Approval.RejectedBy != "" || release.Status.Phase == cicdv1alpha1.ReleasePhaseFailed {
		s.writeError(w, conflict("rejected or failed release cannot be approved"))
		return
	}
	required, err := s.releaseRequiresApproval(r.Context(), release)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if !required {
		s.writeError(w, badRequest("release does not require approval"))
		return
	}
	now := metav1.Now()
	actor := auth.ActorFromContext(r.Context(), req.Actor)
	release.Spec.Approval.Required = true
	release.Spec.Approval.ApprovedBy = actor
	release.Spec.Approval.ApprovedAt = &now
	release.Spec.Approval.RejectedBy = ""
	release.Spec.Approval.RejectedAt = nil
	release.Spec.Approval.Comment = req.Comment
	release.SetAnnotations(mergeStringMap(release.GetAnnotations(), map[string]string{
		"cloudivision.io/approval-action":  "approved",
		"cloudivision.io/approval-actor":   actor,
		"cloudivision.io/approval-comment": req.Comment,
	}))
	if err := s.Client.Update(r.Context(), release); err != nil {
		s.writeError(w, err)
		return
	}
	s.recordAudit(r.Context(), audit.Event{
		Type:     "ReleaseApproved",
		Actor:    actor,
		Project:  release.Spec.ProjectRef,
		BuildRun: release.Spec.BuildRunRef,
		Release:  release.Name,
		Message:  "Release approved.",
		Metadata: auditMetadata(map[string]string{"comment": req.Comment, "namespace": release.Namespace}),
	})
	writeJSON(w, http.StatusOK, releaseDTO(*release))
}

func (s Server) rejectRelease(w http.ResponseWriter, r *http.Request) {
	release, req, ok := s.releaseApprovalActionInput(w, r)
	if !ok {
		return
	}
	if release.Status.Phase == cicdv1alpha1.ReleasePhaseDeployed || release.Status.Phase == cicdv1alpha1.ReleasePhaseDeploying {
		s.writeError(w, conflict("release is already deploying or deployed"))
		return
	}
	if release.Spec.Approval.RejectedBy != "" || release.Status.Phase == cicdv1alpha1.ReleasePhaseFailed {
		s.writeError(w, conflict("release is already rejected or failed"))
		return
	}
	required, err := s.releaseRequiresApproval(r.Context(), release)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if !required {
		s.writeError(w, badRequest("release does not require approval"))
		return
	}
	now := metav1.Now()
	actor := auth.ActorFromContext(r.Context(), req.Actor)
	release.Spec.Approval.Required = true
	release.Spec.Approval.RejectedBy = actor
	release.Spec.Approval.RejectedAt = &now
	release.Spec.Approval.Comment = req.Comment
	release.SetAnnotations(mergeStringMap(release.GetAnnotations(), map[string]string{
		"cloudivision.io/approval-action":  "rejected",
		"cloudivision.io/approval-actor":   actor,
		"cloudivision.io/approval-comment": req.Comment,
	}))
	if err := s.Client.Update(r.Context(), release); err != nil {
		s.writeError(w, err)
		return
	}
	release.Status.Phase = cicdv1alpha1.ReleasePhaseFailed
	release.Status.ObservedGeneration = release.Generation
	release.Status.CompletedAt = &now
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "ReleaseRejected",
		Message:            "Release was rejected by " + actor + ".",
		LastTransitionTime: now,
	})
	if err := s.Client.Status().Update(r.Context(), release); err != nil {
		s.writeError(w, err)
		return
	}
	s.recordAudit(r.Context(), audit.Event{
		Type:     "ReleaseRejected",
		Actor:    actor,
		Project:  release.Spec.ProjectRef,
		BuildRun: release.Spec.BuildRunRef,
		Release:  release.Name,
		Message:  "Release rejected.",
		Metadata: auditMetadata(map[string]string{"comment": req.Comment, "namespace": release.Namespace}),
	})
	writeJSON(w, http.StatusOK, releaseDTO(*release))
}

func (s Server) releaseApprovalActionInput(w http.ResponseWriter, r *http.Request) (*cicdv1alpha1.Release, ReleaseApprovalRequest, bool) {
	var req ReleaseApprovalRequest
	if !s.decode(w, r, &req) {
		return nil, ReleaseApprovalRequest{}, false
	}
	req.Actor = strings.TrimSpace(req.Actor)
	if req.Actor == "" {
		req.Actor = auth.ActorFromContext(r.Context(), "")
	}
	if req.Actor == "" {
		s.writeError(w, badRequest("actor is required when no authenticated principal is available"))
		return nil, ReleaseApprovalRequest{}, false
	}
	release := &cicdv1alpha1.Release{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: r.PathValue("namespace")}, release); err != nil {
		s.writeError(w, err)
		return nil, ReleaseApprovalRequest{}, false
	}
	return release, req, true
}

func (s Server) releaseRequiresApproval(ctx context.Context, release *cicdv1alpha1.Release) (bool, error) {
	if release.Spec.Approval.Required {
		return true, nil
	}
	if release.Spec.EnvironmentRef == "" {
		return false, nil
	}
	environment := &cicdv1alpha1.Environment{}
	if err := s.Client.Get(ctx, client.ObjectKey{Name: release.Spec.EnvironmentRef, Namespace: release.Namespace}, environment); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return environment.Spec.RequiresApproval, nil
}

func (s Server) auditEvents(w http.ResponseWriter, r *http.Request) {
	lister := s.AuditEvents
	if lister == nil {
		if cast, ok := s.Audit.(audit.EventLister); ok {
			lister = cast
		}
	}
	if lister == nil {
		writeJSON(w, http.StatusOK, []AuditEventResponse{})
		return
	}
	events, err := lister.ListEvents(r.Context(), audit.EventFilter{
		Project:  r.URL.Query().Get("project"),
		BuildRun: r.URL.Query().Get("buildRun"),
		Release:  r.URL.Query().Get("release"),
		Type:     r.URL.Query().Get("type"),
	})
	if err != nil {
		s.writeError(w, err)
		return
	}
	items := make([]AuditEventResponse, 0, len(events))
	for _, event := range events {
		items = append(items, auditEventDTO(event))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) webhook(provider webhook.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repositoryName := r.PathValue("repositoryName")
		if err := validateName(repositoryName); err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		namespace := s.namespace(r.URL.Query().Get("namespace"))
		repository := &cicdv1alpha1.Repository{}
		if err := s.Client.Get(r.Context(), client.ObjectKey{Name: repositoryName, Namespace: namespace}, repository); err != nil {
			s.writeError(w, err)
			return
		}
		if !repository.Spec.Webhook.Enabled {
			s.writeError(w, forbidden("webhook is not enabled for repository"))
			return
		}
		if repository.Spec.Provider != cicdv1alpha1.RepositoryProvider(provider) && provider != webhook.ProviderGeneric {
			s.writeError(w, badRequest("webhook provider does not match repository provider"))
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			s.writeError(w, badRequest("request body exceeds limit or cannot be read"))
			return
		}
		defer r.Body.Close()

		secret, err := s.webhookSecret(r.Context(), repository)
		if err != nil {
			s.writeError(w, forbidden(err.Error()))
			return
		}
		if err := webhook.Verify(provider, r.Header, body, secret); err != nil {
			s.writeError(w, unauthorized(err.Error()))
			return
		}
		event, err := webhook.Parse(provider, r.Header, body)
		if err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		if !event.IsPush {
			s.writeError(w, badRequest("only push events are supported"))
			return
		}
		if event.EventID == "" {
			s.writeError(w, badRequest("webhook event ID is required"))
			return
		}
		if event.CommitSHA == "" {
			s.writeError(w, badRequest("webhook commit SHA is required"))
			return
		}
		if event.Branch != repository.Spec.DefaultBranch {
			s.writeError(w, badRequest("webhook branch does not match repository defaultBranch"))
			return
		}

		if existing, ok, err := s.existingBuildRunForEvent(r.Context(), provider, namespace, repository.Name, event.EventID); err != nil {
			s.writeError(w, err)
			return
		} else if ok {
			writeJSON(w, http.StatusOK, WebhookResponse{
				Repository: repository.Name,
				EventID:    event.EventID,
				BuildRun:   buildRunDTO(existing),
				Created:    false,
			})
			return
		}

		project := &cicdv1alpha1.Project{}
		if err := s.Client.Get(r.Context(), client.ObjectKey{Name: repository.Spec.ProjectRef, Namespace: namespace}, project); err != nil {
			s.writeError(w, err)
			return
		}
		template := &cicdv1alpha1.PipelineTemplate{}
		if err := s.Client.Get(r.Context(), client.ObjectKey{Name: repository.Spec.PipelineTemplateRef, Namespace: namespace}, template); err != nil {
			s.writeError(w, err)
			return
		}

		buildRun := buildRunFromWebhook(namespace, repository, project, template, event)
		if err := s.Client.Create(r.Context(), &buildRun); err != nil {
			s.writeError(w, err)
			return
		}
		if err := s.recordWebhookEvent(r.Context(), provider, repository, buildRun, event); err != nil {
			s.writeError(w, err)
			return
		}
		s.recordAudit(r.Context(), audit.Event{
			Type:       "WebhookBuildRunCreated",
			Actor:      event.Actor,
			Project:    repository.Spec.ProjectRef,
			Repository: repository.Name,
			BuildRun:   buildRun.Name,
			EventID:    event.EventID,
			Message:    "Created BuildRun from webhook push event.",
		})
		writeJSON(w, http.StatusCreated, WebhookResponse{
			Repository: repository.Name,
			EventID:    event.EventID,
			BuildRun:   buildRunDTO(buildRun),
			Created:    true,
		})
	}
}

func (s Server) webhookSecret(ctx context.Context, repository *cicdv1alpha1.Repository) (string, error) {
	ref := repository.Spec.Webhook.SecretRef
	if ref.Name == "" || ref.Key == "" {
		return "", fmt.Errorf("webhook secretRef.name and secretRef.key are required")
	}
	secret := &corev1.Secret{}
	if err := s.Client.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: repository.Namespace}, secret); err != nil {
		return "", fmt.Errorf("load webhook secret: %w", err)
	}
	value := secret.Data[ref.Key]
	if len(value) == 0 {
		return "", fmt.Errorf("webhook secret key %q is empty or missing", ref.Key)
	}
	return string(value), nil
}

func (s Server) existingBuildRunForEvent(ctx context.Context, provider webhook.Provider, namespace, repositoryName, eventID string) (cicdv1alpha1.BuildRun, bool, error) {
	if s.WebhookIndex != nil {
		indexed, err := s.WebhookIndex.FindWebhookEvent(ctx, string(provider), repositoryName, eventID)
		if err != nil {
			return cicdv1alpha1.BuildRun{}, false, err
		}
		if indexed != nil && indexed.BuildRun != "" {
			buildRun := cicdv1alpha1.BuildRun{}
			if err := s.Client.Get(ctx, client.ObjectKey{Name: indexed.BuildRun, Namespace: namespace}, &buildRun); err != nil {
				return cicdv1alpha1.BuildRun{}, false, err
			}
			return buildRun, true, nil
		}
	}
	var list cicdv1alpha1.BuildRunList
	if err := s.Client.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return cicdv1alpha1.BuildRun{}, false, err
	}
	for _, item := range list.Items {
		if item.Spec.RepositoryRef == repositoryName && item.Spec.TriggeredBy.EventID == eventID {
			return item, true, nil
		}
	}
	return cicdv1alpha1.BuildRun{}, false, nil
}

func (s Server) recordWebhookEvent(ctx context.Context, provider webhook.Provider, repository *cicdv1alpha1.Repository, buildRun cicdv1alpha1.BuildRun, event webhook.Event) error {
	if s.WebhookIndex == nil {
		return nil
	}
	return s.WebhookIndex.RecordWebhookEvent(ctx, audit.WebhookEvent{
		Provider:   string(provider),
		Repository: repository.Name,
		EventID:    event.EventID,
		Project:    repository.Spec.ProjectRef,
		BuildRun:   buildRun.Name,
	})
}

func (s Server) recordAudit(ctx context.Context, event audit.Event) {
	recorder := s.Audit
	if recorder == nil {
		recorder = audit.LoggerRecorder{Logger: s.Logger}
	}
	if err := recorder.Record(ctx, event); err != nil && s.Logger != nil {
		s.Logger.Warn("record audit event failed", "error", err)
	}
}

func auditMetadata(values map[string]string) json.RawMessage {
	data, err := json.Marshal(values)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(data)
}

func mergeStringMap(base map[string]string, extra map[string]string) map[string]string {
	merged := map[string]string{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func (s Server) list(ctx context.Context, r *http.Request, list client.ObjectList) error {
	if r.URL.Query().Get("allNamespaces") == "true" {
		return s.Client.List(ctx, list)
	}
	return s.Client.List(ctx, list, client.InNamespace(s.namespace(r.URL.Query().Get("namespace"))))
}

func (s Server) decode(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		s.writeError(w, badRequest("invalid JSON body: "+err.Error()))
		return false
	}
	return true
}

func (s Server) namespace(input string) string {
	if input != "" {
		return input
	}
	if s.DefaultNamespace != "" {
		return s.DefaultNamespace
	}
	return "default"
}

func (s Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		permission, protected := auth.RequiredPermission(r.Method, r.URL.Path)
		if !protected {
			if strings.HasPrefix(r.URL.Path, "/api/v1/webhooks/") {
				principal := &auth.Principal{Subject: "webhook", DisplayName: "Webhook", Roles: []auth.Role{auth.RoleDeveloper}}
				next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		authenticator := s.Authenticator
		mode := strings.ToLower(s.AuthMode)
		if mode == "" {
			mode = "disabled"
		}
		if authenticator == nil {
			if mode == "disabled" {
				authenticator = auth.DisabledAuthenticator{}
			} else {
				writeError(w, http.StatusNotImplemented, "auth_not_configured", "authentication provider is not configured")
				return
			}
		}
		principal, err := authenticator.Authenticate(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		if principal.DevMode {
			w.Header().Set("X-Cloudivision-Auth-Mode", "development")
		}
		if !auth.Allowed(principal, permission) {
			writeError(w, http.StatusForbidden, "forbidden", "principal does not have permission "+string(permission))
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
	})
}

func (s Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(observability): extract and propagate W3C trace context here when OpenTelemetry is introduced.
		requestID := observability.RequestIDFromRequest(r)
		w.Header().Set(observability.RequestIDHeader, requestID)
		ctx := observability.WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s Server) originAllowed(origin string) bool {
	for _, allowed := range s.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func (s Server) logging(next http.Handler) http.Handler {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		route := routeLabel(r)
		observability.ObserveHTTPRequest(r.Method, route, recorder.status, started)
		logger.Info(
			"api request",
			"method", r.Method,
			"route", route,
			"path", redact.MaskString(r.URL.RequestURI()),
			"status", recorder.status,
			"durationMs", time.Since(started).Milliseconds(),
			"requestId", observability.RequestIDFromContext(r.Context()),
		)
	})
}

func (s Server) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := err.Error()
	var apiErr apiError
	if errors.As(err, &apiErr) {
		status = apiErr.status
		code = apiErr.code
		message = apiErr.message
	} else if apierrors.IsNotFound(err) {
		status = http.StatusNotFound
		code = "not_found"
	} else if apierrors.IsAlreadyExists(err) {
		status = http.StatusConflict
		code = "already_exists"
	}
	writeError(w, status, code, message)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSONStatus(w, status, ErrorResponse{
		Code:      code,
		Message:   redact.MaskString(message),
		RequestID: w.Header().Get(observability.RequestIDHeader),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	writeJSONStatus(w, status, value)
}

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func routeLabel(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}

type apiError struct {
	status  int
	code    string
	message string
}

func (e apiError) Error() string {
	return e.message
}

func badRequest(message string) error {
	return apiError{status: http.StatusBadRequest, code: "bad_request", message: message}
}

func notFound(message string) error {
	return apiError{status: http.StatusNotFound, code: "not_found", message: message}
}

func unauthorized(message string) error {
	return apiError{status: http.StatusUnauthorized, code: "unauthorized", message: message}
}

func forbidden(message string) error {
	return apiError{status: http.StatusForbidden, code: "forbidden", message: message}
}

func conflict(message string) error {
	return apiError{status: http.StatusConflict, code: "conflict", message: message}
}

func internalError(message string) error {
	return apiError{status: http.StatusInternalServerError, code: "internal_error", message: message}
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func validateBuildRunRequest(req BuildRunRequest) error {
	if err := validateName(req.Name); err != nil {
		return err
	}
	if req.Spec.ProjectRef == "" || req.Spec.RepositoryRef == "" || req.Spec.PipelineTemplateRef == "" {
		return fmt.Errorf("projectRef, repositoryRef and pipelineTemplateRef are required")
	}
	if req.Spec.Revision == "" && req.Spec.Branch == "" {
		return fmt.Errorf("revision or branch is required")
	}
	if req.Spec.TriggeredBy.Type == "" {
		return fmt.Errorf("triggeredBy.type is required")
	}
	if req.Spec.Image.Repository == "" {
		return fmt.Errorf("image.repository is required")
	}
	return nil
}

func parseTailLines(value string) (*int64, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return nil, fmt.Errorf("tailLines must be a non-negative integer")
	}
	return &parsed, nil
}

func splitLogLines(logs string) []string {
	trimmed := strings.TrimRight(logs, "\n")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "\n")
}

func buildRunFromWebhook(namespace string, repository *cicdv1alpha1.Repository, project *cicdv1alpha1.Project, template *cicdv1alpha1.PipelineTemplate, event webhook.Event) cicdv1alpha1.BuildRun {
	imageRepository := template.Spec.Build.Image
	if imageRepository == "" {
		imageRepository = strings.TrimRight(project.Spec.DefaultRegistry, "/") + "/" + repository.Name
	}
	tag := shortSHA(event.CommitSHA)
	if tag == "" {
		tag = event.Branch
	}
	return cicdv1alpha1.BuildRun{
		ObjectMeta: objectMeta(buildRunNameForWebhook(repository.Name, event.EventID), namespace),
		Spec: cicdv1alpha1.BuildRunSpec{
			ProjectRef:          repository.Spec.ProjectRef,
			RepositoryRef:       repository.Name,
			PipelineTemplateRef: repository.Spec.PipelineTemplateRef,
			Revision:            event.CommitSHA,
			Branch:              event.Branch,
			CommitSHA:           event.CommitSHA,
			TriggeredBy: cicdv1alpha1.TriggeredBy{
				Type:    cicdv1alpha1.TriggerTypeWebhook,
				Actor:   event.Actor,
				EventID: event.EventID,
			},
			Image: cicdv1alpha1.ImageRef{
				Repository: imageRepository,
				Tag:        tag,
			},
			Executor: cicdv1alpha1.ExecutorTypeJob,
		},
	}
}

func buildRunNameForWebhook(repositoryName, eventID string) string {
	hash := sha256.Sum256([]byte(eventID))
	base := dnsLabel(repositoryName)
	if base == "" {
		base = "repository"
	}
	name := fmt.Sprintf("%s-%x", base, hash[:6])
	if len(name) > 63 {
		return name[:63]
	}
	return name
}

func dnsLabel(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func shortSHA(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

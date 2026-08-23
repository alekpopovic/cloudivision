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
	"github.com/cloudivision/cloudivision/internal/artifacts"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/auth"
	buildlogic "github.com/cloudivision/cloudivision/internal/build"
	dependencycache "github.com/cloudivision/cloudivision/internal/cache"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/kube"
	"github.com/cloudivision/cloudivision/internal/logstore"
	"github.com/cloudivision/cloudivision/internal/observability"
	"github.com/cloudivision/cloudivision/internal/policy"
	"github.com/cloudivision/cloudivision/internal/provider"
	providernotifications "github.com/cloudivision/cloudivision/internal/provider/notifications"
	providersecrets "github.com/cloudivision/cloudivision/internal/provider/secrets"
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
	LogStore         logstore.LogStore
	ArtifactStore    artifacts.ArtifactStore
	CacheStore       dependencycache.Store
	Logger           *slog.Logger
	Audit            audit.Recorder
	AuditEvents      audit.EventLister
	WebhookIndex     audit.WebhookIndexer
	Authenticator    auth.Authenticator
	DefaultNamespace string
	AuthMode         string
	CORSOrigins      []string
	MetricsEnabled   bool
	Providers        *provider.Registry
	PolicyEvaluator  policy.Evaluator
	Notifier         providernotifications.Dispatcher
	Organizations    auth.OrganizationDirectory
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.health)
	if s.MetricsEnabled {
		mux.Handle("GET /metrics", observability.MetricsHandler())
	}
	mux.HandleFunc("GET /api/v1/auth/me", s.currentUser)
	mux.HandleFunc("GET /api/v1/organizations", s.organizations)
	mux.HandleFunc("GET /api/v1/organizations/{organization}/teams", s.organizationTeams)
	mux.HandleFunc("GET /api/v1/organizations/{organization}/members", s.organizationMembers)
	mux.HandleFunc("GET /api/v1/organizations/{organization}/project-access", s.organizationProjectAccess)
	mux.HandleFunc("GET /api/v1/projects", s.projects)
	mux.HandleFunc("POST /api/v1/projects", s.projects)
	mux.HandleFunc("GET /api/v1/projects/{name}", s.project)
	mux.HandleFunc("POST /api/v1/projects/{name}/cache/purge", s.purgeProjectCache)
	mux.HandleFunc("GET /api/v1/repositories", s.repositories)
	mux.HandleFunc("POST /api/v1/repositories", s.repositories)
	mux.HandleFunc("GET /api/v1/pipeline-templates", s.pipelineTemplates)
	mux.HandleFunc("POST /api/v1/pipeline-templates", s.pipelineTemplates)
	mux.HandleFunc("PUT /api/v1/pipeline-templates/{namespace}/{name}", s.updatePipelineTemplate)
	mux.HandleFunc("GET /api/v1/catalog/pipeline-templates", s.catalogPipelineTemplates)
	mux.HandleFunc("POST /api/v1/catalog/pipeline-templates/{name}/install", s.installCatalogPipelineTemplate)
	mux.HandleFunc("GET /api/v1/build-runs", s.buildRuns)
	mux.HandleFunc("POST /api/v1/build-runs", s.buildRuns)
	mux.HandleFunc("GET /api/v1/build-runs/{namespace}/{name}", s.buildRun)
	mux.HandleFunc("POST /api/v1/build-runs/{namespace}/{name}/cancel", s.cancelBuildRun)
	mux.HandleFunc("POST /api/v1/build-runs/{namespace}/{name}/retry", s.retryBuildRun)
	mux.HandleFunc("POST /api/v1/build-runs/{namespace}/{name}/rerun", s.rerunBuildRun)
	mux.HandleFunc("GET /api/v1/build-runs/{namespace}/{name}/logs", s.buildRunLogs)
	mux.HandleFunc("GET /api/v1/build-runs/{namespace}/{name}/artifacts", s.buildRunArtifacts)
	mux.HandleFunc("GET /api/v1/build-runs/{namespace}/{name}/artifacts/{artifactName}", s.buildRunArtifact)
	mux.HandleFunc("GET /api/v1/environments", s.environments)
	mux.HandleFunc("GET /api/v1/cluster-targets", s.clusterTargets)
	mux.HandleFunc("GET /api/v1/releases", s.releases)
	mux.HandleFunc("POST /api/v1/releases/{namespace}/{name}/approve", s.approveRelease)
	mux.HandleFunc("POST /api/v1/releases/{namespace}/{name}/reject", s.rejectRelease)
	mux.HandleFunc("POST /api/v1/releases/{namespace}/{name}/promote", s.promoteRelease)
	mux.HandleFunc("POST /api/v1/releases/{namespace}/{name}/rollback", s.rollbackRelease)
	mux.HandleFunc("GET /api/v1/audit/events", s.auditEvents)
	mux.HandleFunc("GET /api/v1/audit/events/export", s.auditExport)
	mux.HandleFunc("GET /api/v1/reports/builds", s.buildReport)
	mux.HandleFunc("GET /api/v1/reports/releases", s.releaseReport)
	mux.HandleFunc("GET /api/v1/reports/security", s.securityReport)
	mux.HandleFunc("GET /api/v1/providers", s.providers)
	mux.HandleFunc("GET /api/v1/providers/health", s.providerHealth)
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

func (s Server) organizations(w http.ResponseWriter, r *http.Request) {
	if s.Organizations == nil {
		writeJSON(w, http.StatusOK, []auth.Organization{})
		return
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	items, err := s.Organizations.ListOrganizations(r.Context(), principal.Subject)
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) organizationTeams(w http.ResponseWriter, r *http.Request) {
	if !s.requireOrganizationRead(w, r) {
		return
	}
	items, err := s.Organizations.ListTeams(r.Context(), r.PathValue("organization"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) organizationMembers(w http.ResponseWriter, r *http.Request) {
	if !s.requireOrganizationRead(w, r) {
		return
	}
	items, err := s.Organizations.ListMemberships(r.Context(), r.PathValue("organization"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) organizationProjectAccess(w http.ResponseWriter, r *http.Request) {
	if !s.requireOrganizationRead(w, r) {
		return
	}
	items, err := s.Organizations.ListProjectAccess(r.Context(), r.PathValue("organization"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (s Server) requireOrganizationRead(w http.ResponseWriter, r *http.Request) bool {
	if s.Organizations == nil {
		s.writeError(w, apiError{status: http.StatusServiceUnavailable, code: "organizations_unavailable", message: "organization directory is not configured"})
		return false
	}
	principal, _ := auth.PrincipalFromContext(r.Context())
	allowed, err := s.Organizations.Allowed(r.Context(), principal, r.PathValue("organization"), "", auth.PermissionRead)
	if err != nil {
		s.writeError(w, err)
		return false
	}
	if !allowed {
		s.writeError(w, apiError{status: http.StatusForbidden, code: "forbidden", message: "organization access denied"})
		return false
	}
	return true
}

func (s Server) providers(w http.ResponseWriter, _ *http.Request) {
	if s.Providers == nil {
		writeJSON(w, http.StatusOK, []provider.Summary{})
		return
	}
	writeJSON(w, http.StatusOK, s.Providers.Summaries())
}

func (s Server) providerHealth(w http.ResponseWriter, r *http.Request) {
	if s.Providers == nil {
		writeJSON(w, http.StatusOK, []provider.HealthResult{})
		return
	}
	writeJSON(w, http.StatusOK, s.Providers.HealthCheckAll(r.Context()))
}

func (s Server) clusterTargets(w http.ResponseWriter, r *http.Request) {
	var list cicdv1alpha1.ClusterTargetList
	if err := s.list(r.Context(), r, &list); err != nil {
		s.writeError(w, err)
		return
	}
	items := make([]ClusterTargetResponse, 0, len(list.Items)+1)
	items = append(items, ClusterTargetResponse{
		Name:      "local",
		Namespace: s.namespace(r.URL.Query().Get("namespace")),
		Spec:      cicdv1alpha1.ClusterTargetSpec{DisplayName: "Local cluster", Type: cicdv1alpha1.ClusterTargetTypeBoth},
		Status:    cicdv1alpha1.ClusterTargetStatus{Phase: cicdv1alpha1.ClusterTargetPhaseReady, Reachable: true},
	})
	for _, item := range list.Items {
		items = append(items, clusterTargetDTO(item))
	}
	writeJSON(w, http.StatusOK, items)
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

func (s Server) updatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	var request PipelineTemplateRequest
	if !s.decode(w, r, &request) {
		return
	}
	if err := validatePipelineTemplateSpec(request.Spec); err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	object := &cicdv1alpha1.PipelineTemplate{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: r.PathValue("namespace"), Name: r.PathValue("name")}, object); err != nil {
		s.writeError(w, err)
		return
	}
	object.Spec = request.Spec
	if err := s.Client.Update(r.Context(), object); err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pipelineTemplateDTO(*object))
}

func validatePipelineTemplateSpec(spec cicdv1alpha1.PipelineTemplateSpec) error {
	if len(spec.Steps) == 0 && !spec.Build.Enabled {
		return errors.New("at least one step or an enabled build is required")
	}
	names := map[string]bool{}
	for index, step := range spec.Steps {
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("steps[%d].name is required", index)
		}
		if strings.TrimSpace(step.Image) == "" {
			return fmt.Errorf("steps[%d].image is required", index)
		}
		if len(step.Command) == 0 {
			return fmt.Errorf("steps[%d].command is required", index)
		}
		if names[step.Name] {
			return fmt.Errorf("duplicate step name %q", step.Name)
		}
		names[step.Name] = true
	}
	if spec.Build.Enabled && spec.Build.Builder == "" {
		return errors.New("build.builder is required when build.enabled is true")
	}
	return nil
}

func (s Server) project(w http.ResponseWriter, r *http.Request) {
	var obj cicdv1alpha1.Project
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: s.namespace(r.URL.Query().Get("namespace"))}, &obj); err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, projectDTO(obj))
}

func (s Server) purgeProjectCache(w http.ResponseWriter, r *http.Request) {
	project := &cicdv1alpha1.Project{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: s.namespace(r.URL.Query().Get("namespace"))}, project); err != nil {
		s.writeError(w, err)
		return
	}
	if s.CacheStore == nil {
		s.writeError(w, apiError{status: http.StatusServiceUnavailable, code: "cache_unavailable", message: "dependency cache backend is not configured"})
		return
	}
	repository := strings.TrimSpace(r.URL.Query().Get("repository"))
	if repository != "" {
		repositoryObject := &cicdv1alpha1.Repository{}
		if err := s.Client.Get(r.Context(), client.ObjectKey{Name: repository, Namespace: project.Namespace}, repositoryObject); err != nil {
			s.writeError(w, err)
			return
		}
		if repositoryObject.Spec.ProjectRef != project.Name {
			s.writeError(w, badRequest("repository does not belong to project"))
			return
		}
	}
	if err := s.CacheStore.Purge(r.Context(), dependencycache.PurgeRequest{Project: project.Name, Repository: repository}); err != nil {
		s.writeError(w, fmt.Errorf("purge project dependency cache: %w", err))
		return
	}
	s.recordAudit(r.Context(), audit.Event{Type: "ProjectCachePurged", Actor: auth.ActorFromContext(r.Context(), ""), Project: project.Name, Repository: repository, Message: "Purged dependency cache."})
	writeJSON(w, http.StatusOK, CachePurgeResponse{Project: project.Name, Repository: repository, Purged: true})
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
		if err := validatePipelineTemplateSpec(req.Spec); err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		if req.Spec.Build.Enabled && req.Spec.Build.Builder == "" {
			s.writeError(w, badRequest("build.builder is required when build.enabled is true"))
			return
		}
		if req.Spec.Cache.Enabled {
			if req.Spec.Cache.Mode == "" {
				s.writeError(w, badRequest("cache.mode is required when cache.enabled is true"))
				return
			}
			if req.Spec.Cache.Mode == cicdv1alpha1.DependencyCacheModePVC && len(req.Spec.Cache.Paths) == 0 {
				s.writeError(w, badRequest("cache.paths is required for pvc cache mode"))
				return
			}
			if req.Spec.Cache.Mode == cicdv1alpha1.DependencyCacheModeRegistry && (!req.Spec.Build.Enabled || req.Spec.Build.Builder != cicdv1alpha1.BuildBuilderBuildKit) {
				s.writeError(w, badRequest("registry cache mode requires an enabled BuildKit image build"))
				return
			}
		}
		obj := &cicdv1alpha1.PipelineTemplate{ObjectMeta: objectMeta(req.Name, s.namespace(req.Namespace)), Spec: req.Spec}
		policyEvaluator := s.PolicyEvaluator
		if policyEvaluator == nil {
			policyEvaluator = policy.NewDefaultEvaluator()
		}
		decision := policyEvaluator.EvaluatePipelineTemplate(r.Context(), policy.PipelineTemplatePolicyInput{
			PipelineTemplate: obj, EnforceAuthorization: true, SecretUseAuthorized: true,
		})
		if !decision.Allowed {
			s.writeError(w, policyDenied(decision))
			return
		}
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
		if err := s.listBuildRuns(r.Context(), r, &list); err != nil {
			s.writeError(w, err)
			return
		}
		options, err := parsePageOptions(r, map[string]bool{"createdAt": true, "name": true, "phase": true})
		if err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		filtered := filterSortBuildRuns(r, list.Items, options)
		page, next := paginate(filtered, options)
		items := make([]BuildRunResponse, 0, len(page))
		for _, item := range page {
			items = append(items, buildRunDTO(item))
		}
		writeJSON(w, http.StatusOK, PageResponse[BuildRunResponse]{Items: items, NextPageToken: next, TotalCount: len(filtered), Limit: options.limit})
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
		policyEvaluator := s.PolicyEvaluator
		if policyEvaluator == nil {
			policyEvaluator = policy.NewDefaultEvaluator()
		}
		decision := policyEvaluator.EvaluateBuildRun(r.Context(), policy.BuildRunPolicyInput{
			BuildRun: obj, EnforceAuthorization: true, TriggerAuthorized: true, SecretUseAuthorized: true,
		})
		if !decision.Allowed {
			s.writeError(w, policyDenied(decision))
			return
		}
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
	if s.LogStore != nil {
		tail := 0
		if tailLines != nil {
			tail = int(*tailLines)
		}
		result, readErr := s.LogStore.Read(r.Context(), logstore.ReadLogRequest{Namespace: namespace, BuildRun: name, TailLines: tail, Step: r.URL.Query().Get("step")})
		if readErr != nil {
			if errors.Is(readErr, logstore.ErrNotFound) {
				s.writeError(w, notFound("stored logs not found"))
			} else {
				s.writeError(w, fmt.Errorf("read stored logs: %w", readErr))
			}
			return
		}
		lines := make([]string, 0, len(result.Lines))
		for _, line := range result.Lines {
			prefix := ""
			if !line.Timestamp.IsZero() {
				prefix = line.Timestamp.UTC().Format(time.RFC3339Nano) + " "
			}
			if line.Step != "" {
				prefix += "[step:" + line.Step + "] "
			}
			lines = append(lines, prefix+line.Message)
		}
		writeJSON(w, http.StatusOK, LogsResponse{Namespace: namespace, BuildRun: name, Backend: result.Backend, Ref: result.Ref, Lines: lines})
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
	options, err := parsePageOptions(r, map[string]bool{"createdAt": true, "name": true, "phase": true})
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	filtered := filterSortReleases(r, list.Items, options)
	page, next := paginate(filtered, options)
	items := make([]ReleaseResponse, 0, len(page))
	for _, item := range page {
		items = append(items, releaseDTO(item))
	}
	writeJSON(w, http.StatusOK, PageResponse[ReleaseResponse]{Items: items, NextPageToken: next, TotalCount: len(filtered), Limit: options.limit})
}

func (s Server) approveRelease(w http.ResponseWriter, r *http.Request) {
	release, req, ok := s.releaseApprovalActionInput(w, r)
	if !ok {
		return
	}
	if releasePromotionStarted(release.Status.Phase) {
		s.writeError(w, conflict("release is already deploying or deployed"))
		return
	}
	if release.Spec.Approval.RejectedBy != "" || releasePhaseFailed(release.Status.Phase) {
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
	if releasePromotionStarted(release.Status.Phase) {
		s.writeError(w, conflict("release is already deploying or deployed"))
		return
	}
	if release.Spec.Approval.RejectedBy != "" || releasePhaseFailed(release.Status.Phase) {
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
	release.Status.Phase = cicdv1alpha1.ReleasePhaseFailedApproval
	release.Status.ObservedGeneration = release.Generation
	release.Status.CompletedAt = &now
	release.Status.Approval = cicdv1alpha1.ReleaseApprovalStatus{RejectedBy: actor, RejectedAt: &now}
	release.Status.Failure = cicdv1alpha1.FailureStatus{Reason: "ReleaseRejected", Message: "Release was rejected by " + actor + "."}
	domain.SetCondition(&release.Status.Conditions, metav1.Condition{
		Type:               domain.ConditionFailed,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: release.Generation,
		Reason:             "ReleaseRejected",
		Message:            "Release was rejected by " + actor + ".",
		LastTransitionTime: now,
	})
	if err := kube.UpdateStatusWithRetry(r.Context(), s.Client, release); err != nil {
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

func releasePromotionStarted(phase cicdv1alpha1.ReleasePhase) bool {
	switch phase {
	case cicdv1alpha1.ReleasePhasePreparingGitOpsChange, cicdv1alpha1.ReleasePhaseGitOpsChangeCommitted,
		cicdv1alpha1.ReleasePhaseWaitingForSync, cicdv1alpha1.ReleasePhaseDeployed,
		cicdv1alpha1.ReleasePhaseRolledBack:
		return true
	default:
		return false
	}
}

func releasePhaseFailed(phase cicdv1alpha1.ReleasePhase) bool {
	switch phase {
	case cicdv1alpha1.ReleasePhaseFailedValidation, cicdv1alpha1.ReleasePhaseFailedApproval,
		cicdv1alpha1.ReleasePhaseFailedGitClone, cicdv1alpha1.ReleasePhaseFailedGitCommit,
		cicdv1alpha1.ReleasePhaseFailedGitPush, cicdv1alpha1.ReleasePhaseFailedProviderStatus,
		cicdv1alpha1.ReleasePhaseTimedOut:
		return true
	default:
		return false
	}
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
	options, err := parsePageOptions(r, map[string]bool{"createdAt": true, "type": true})
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	lister := s.AuditEvents
	if lister == nil {
		if cast, ok := s.Audit.(audit.EventLister); ok {
			lister = cast
		}
	}
	if lister == nil {
		writeJSON(w, http.StatusOK, PageResponse[AuditEventResponse]{Items: []AuditEventResponse{}, Limit: options.limit})
		return
	}
	events, err := lister.ListEvents(r.Context(), audit.EventFilter{
		Organization: r.URL.Query().Get("organization"),
		Project:      r.URL.Query().Get("project"),
		Repository:   r.URL.Query().Get("repository"),
		BuildRun:     r.URL.Query().Get("buildRun"),
		Release:      r.URL.Query().Get("release"),
		Type:         r.URL.Query().Get("type"),
		Actor:        r.URL.Query().Get("actor"),
		From:         options.from,
		To:           options.to,
	})
	if err != nil {
		s.writeError(w, err)
		return
	}
	filtered := filterSortAuditEvents(events, options)
	page, next := paginate(filtered, options)
	items := make([]AuditEventResponse, 0, len(page))
	for _, event := range page {
		items = append(items, auditEventDTO(event))
	}
	writeJSON(w, http.StatusOK, PageResponse[AuditEventResponse]{Items: items, NextPageToken: next, TotalCount: len(filtered), Limit: options.limit})
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
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, webhook.DeliveryID(provider, r.Header), "", "webhook_disabled", "", "")
			s.writeError(w, forbidden("webhook is not enabled for repository"))
			return
		}
		if repository.Spec.Provider != cicdv1alpha1.RepositoryProvider(provider) && provider != webhook.ProviderGeneric {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, webhook.DeliveryID(provider, r.Header), "", "provider_mismatch", "", "")
			s.writeError(w, badRequest("webhook provider does not match repository provider"))
			return
		}
		deliveryID := webhook.DeliveryID(provider, r.Header)
		defer r.Body.Close()
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, deliveryID, "", "body_too_large", "", "")
			observability.WebhookEvents.WithLabelValues(string(provider), "body_too_large").Inc()
			s.writeError(w, apiError{status: http.StatusRequestEntityTooLarge, code: "payload_too_large", message: "webhook payload exceeds the 1 MiB limit"})
			return
		}
		secret, err := s.webhookSecret(r.Context(), repository)
		if err != nil {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, deliveryID, "", "secret_unavailable", "", "")
			s.writeError(w, forbidden(err.Error()))
			return
		}
		if err := webhook.Verify(provider, r.Header, body, secret); err != nil {
			observability.WebhookEvents.WithLabelValues(string(provider), "invalid_signature").Inc()
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, deliveryID, "", "signature_verification_failed", "", "")
			s.writeError(w, unauthorized("webhook signature verification failed"))
			return
		}
		event, err := webhook.Parse(provider, r.Header, body)
		if err != nil {
			observability.WebhookEvents.WithLabelValues(string(provider), "invalid_payload").Inc()
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, deliveryID, "", "malformed_payload", "", "")
			s.writeError(w, badRequest("malformed webhook payload"))
			return
		}
		if event.EventID == "" {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, "", string(event.Type), "missing_delivery_id", event.Actor, "")
			s.writeError(w, badRequest("webhook event ID is required"))
			return
		}
		if len(event.EventID) > 255 {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, "", string(event.Type), "invalid_delivery_id", event.Actor, "")
			s.writeError(w, badRequest("webhook event ID exceeds 255 characters"))
			return
		}
		if provider == webhook.ProviderGitHub && !event.Timestamp.IsZero() {
			now := time.Now().UTC()
			if event.Timestamp.Before(now.Add(-24*time.Hour)) || event.Timestamp.After(now.Add(5*time.Minute)) {
				observability.WebhookEvents.WithLabelValues(string(provider), "replay_rejected").Inc()
				s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, event.EventID, string(event.Type), "event_timestamp_outside_replay_window", event.Actor, "")
				s.writeError(w, unauthorized("webhook event timestamp is outside the replay window"))
				return
			}
		}

		if existing, ok, err := s.existingBuildRunForEvent(r.Context(), provider, namespace, repository.Name, event.EventID); err != nil {
			s.writeError(w, err)
			return
		} else if ok {
			observability.WebhookEvents.WithLabelValues(string(provider), "duplicate").Inc()
			buildRun := optionalBuildRunDTO(existing)
			s.recordWebhookAudit(r.Context(), "WebhookDuplicate", provider, repository, event.EventID, string(event.Type), "delivery_id_already_processed", event.Actor, existing.Name)
			writeJSON(w, http.StatusOK, WebhookResponse{
				Repository: repository.Name,
				EventID:    event.EventID,
				Event:      string(event.Type),
				Result:     "duplicate",
				Message:    "Webhook delivery was already processed.",
				BuildRun:   buildRun,
				Created:    false,
			})
			return
		}

		if event.IsPing {
			if err := s.recordWebhookEvent(r.Context(), provider, repository, "", event); err != nil {
				s.writeError(w, err)
				return
			}
			s.recordWebhookAudit(r.Context(), "WebhookAccepted", provider, repository, event.EventID, string(event.Type), "ping", event.Actor, "")
			observability.WebhookEvents.WithLabelValues(string(provider), "accepted").Inc()
			writeJSON(w, http.StatusOK, WebhookResponse{Repository: repository.Name, EventID: event.EventID, Event: string(event.Type), Result: "accepted", Message: "GitHub ping accepted.", Created: false})
			return
		}

		matched, filterReason, filterMessage := webhook.Filter(repository, event)
		if !matched {
			if err := s.recordWebhookEvent(r.Context(), provider, repository, "", event); err != nil {
				s.writeError(w, err)
				return
			}
			s.recordWebhookAudit(r.Context(), "WebhookAccepted", provider, repository, event.EventID, string(event.Type), filterReason, event.Actor, "")
			observability.WebhookEvents.WithLabelValues(string(provider), "ignored").Inc()
			writeJSON(w, http.StatusOK, WebhookResponse{Repository: repository.Name, EventID: event.EventID, Event: string(event.Type), Result: "ignored", Message: filterMessage, Created: false})
			return
		}
		if event.CommitSHA == "" {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, event.EventID, string(event.Type), "missing_commit_sha", event.Actor, "")
			s.writeError(w, badRequest("webhook commit SHA is required"))
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

		buildRun, err := buildRunFromWebhook(namespace, repository, project, template, event)
		if err != nil {
			s.writeError(w, badRequest(err.Error()))
			return
		}
		policyEvaluator := s.PolicyEvaluator
		if policyEvaluator == nil {
			policyEvaluator = policy.NewDefaultEvaluator()
		}
		decision := policyEvaluator.EvaluateBuildRun(r.Context(), policy.BuildRunPolicyInput{
			BuildRun: &buildRun, Project: project, Repository: repository, PipelineTemplate: template,
			EnforceAuthorization: true, TriggerAuthorized: true, SecretUseAuthorized: true,
		})
		if !decision.Allowed {
			s.recordWebhookAudit(r.Context(), "WebhookRejected", provider, repository, event.EventID, string(event.Type), "policy_denied", event.Actor, "")
			s.writeError(w, policyDenied(decision))
			return
		}
		if err := s.Client.Create(r.Context(), &buildRun); err != nil {
			if apierrors.IsAlreadyExists(err) {
				existing := cicdv1alpha1.BuildRun{}
				if getErr := s.Client.Get(r.Context(), client.ObjectKeyFromObject(&buildRun), &existing); getErr == nil {
					s.recordWebhookAudit(r.Context(), "WebhookDuplicate", provider, repository, event.EventID, string(event.Type), "build_run_already_exists", event.Actor, existing.Name)
					response := buildRunDTO(existing)
					writeJSON(w, http.StatusOK, WebhookResponse{Repository: repository.Name, EventID: event.EventID, Event: string(event.Type), Result: "duplicate", Message: "Webhook delivery was already processed.", BuildRun: &response, Created: false})
					return
				}
			}
			s.writeError(w, err)
			return
		}
		if err := s.recordWebhookEvent(r.Context(), provider, repository, buildRun.Name, event); err != nil {
			observability.AuditWriteFailures.Inc()
			s.writeError(w, err)
			return
		}
		s.recordWebhookAudit(r.Context(), "WebhookAccepted", provider, repository, event.EventID, string(event.Type), "build_run_created", event.Actor, buildRun.Name)
		s.recordWebhookAudit(r.Context(), "BuildRunCreatedFromWebhook", provider, repository, event.EventID, string(event.Type), "build_run_created", event.Actor, buildRun.Name)
		observability.WebhookEvents.WithLabelValues(string(provider), "accepted").Inc()
		response := buildRunDTO(buildRun)
		writeJSON(w, http.StatusCreated, WebhookResponse{
			Repository: repository.Name,
			EventID:    event.EventID,
			Event:      string(event.Type),
			Result:     "created",
			Message:    "BuildRun created from webhook.",
			BuildRun:   &response,
			Created:    true,
		})
	}
}

func (s Server) webhookSecret(ctx context.Context, repository *cicdv1alpha1.Repository) (string, error) {
	ref := repository.Spec.Webhook.SecretRef
	if ref.Name == "" || ref.Key == "" {
		return "", fmt.Errorf("webhook secretRef.name and secretRef.key are required")
	}
	resolver := providersecrets.KubernetesProvider{Client: s.Client, AllowedNamespaces: []string{repository.Namespace}}
	secret, err := resolver.Resolve(ctx, providersecrets.SecretRef{Namespace: repository.Namespace, Name: ref.Name, Keys: []string{ref.Key}})
	if err != nil {
		return "", fmt.Errorf("load webhook secret: %w", err)
	}
	value := secret.Values[ref.Key]
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
				if apierrors.IsNotFound(err) {
					return cicdv1alpha1.BuildRun{}, true, nil
				}
				return cicdv1alpha1.BuildRun{}, false, err
			}
			return buildRun, true, nil
		}
		if indexed != nil {
			return cicdv1alpha1.BuildRun{}, true, nil
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

func (s Server) recordWebhookEvent(ctx context.Context, provider webhook.Provider, repository *cicdv1alpha1.Repository, buildRunName string, event webhook.Event) error {
	if s.WebhookIndex == nil {
		return nil
	}
	return s.WebhookIndex.RecordWebhookEvent(ctx, audit.WebhookEvent{
		Provider:   string(provider),
		Repository: repository.Name,
		EventID:    event.EventID,
		Project:    repository.Spec.ProjectRef,
		BuildRun:   buildRunName,
	})
}

func optionalBuildRunDTO(buildRun cicdv1alpha1.BuildRun) *BuildRunResponse {
	if buildRun.Name == "" {
		return nil
	}
	response := buildRunDTO(buildRun)
	return &response
}

func (s Server) recordWebhookAudit(ctx context.Context, eventType string, provider webhook.Provider, repository *cicdv1alpha1.Repository, eventID, webhookEvent, reason, actor, buildRun string) {
	message := map[string]string{
		"WebhookAccepted":            "Webhook delivery accepted.",
		"WebhookRejected":            "Webhook delivery rejected.",
		"WebhookDuplicate":           "Duplicate webhook delivery ignored.",
		"BuildRunCreatedFromWebhook": "BuildRun created from webhook.",
	}[eventType]
	s.recordAudit(ctx, audit.Event{
		Type:       eventType,
		Actor:      actor,
		Project:    repository.Spec.ProjectRef,
		Repository: repository.Name,
		BuildRun:   buildRun,
		EventID:    eventID,
		Message:    message,
		Metadata: auditMetadata(map[string]string{
			"provider":   string(provider),
			"event":      webhookEvent,
			"reason":     reason,
			"deliveryID": eventID,
		}),
	})
	if eventType == "WebhookRejected" && s.Notifier != nil && repository != nil {
		_ = s.Notifier.Notify(ctx, providernotifications.NotificationRequest{Event: providernotifications.WebhookRejected, Project: repository.Spec.ProjectRef, Repository: repository.Name, Namespace: repository.Namespace, ResourceName: repository.Name, Message: "Webhook delivery rejected.", OccurredAt: time.Now().UTC()})
	}
}

func (s Server) recordAudit(ctx context.Context, event audit.Event) {
	if event.Organization == "" {
		event.Organization = auth.OrganizationFromContext(ctx)
	}
	recorder := s.Audit
	if recorder == nil {
		recorder = audit.LoggerRecorder{Logger: s.Logger}
	}
	if err := recorder.Record(ctx, event); err != nil {
		observability.AuditWriteFailures.Inc()
		if s.Logger != nil {
			s.Logger.Warn("record audit event failed", "error", err)
		}
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

func (s Server) listBuildRuns(ctx context.Context, r *http.Request, list *cicdv1alpha1.BuildRunList) error {
	namespace := r.URL.Query().Get("namespace")
	if r.URL.Query().Get("allNamespaces") == "true" && namespace == "" {
		return s.Client.List(ctx, list)
	}
	return s.Client.List(ctx, list, client.InNamespace(s.namespace(namespace)))
}

func boundedInt(value string, fallback, minimum, maximum int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("must be an integer from %d to %d", minimum, maximum)
	}
	return parsed, nil
}

func optionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("must be RFC3339")
	}
	return &parsed, nil
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
		organization := strings.TrimSpace(r.Header.Get("X-Cloudivision-Organization"))
		if organization == "" && principal.DevMode {
			organization = "default"
		}
		if organization != "" && s.Organizations != nil && !principal.DevMode {
			allowed, directoryErr := s.Organizations.Allowed(r.Context(), principal, organization, strings.TrimSpace(r.Header.Get("X-Cloudivision-Project")), permission)
			if directoryErr != nil {
				s.writeError(w, directoryErr)
				return
			}
			if !allowed {
				writeError(w, http.StatusForbidden, "forbidden", "organization membership does not grant permission "+string(permission))
				return
			}
		}
		ctx := auth.WithPrincipal(r.Context(), principal)
		ctx = auth.WithOrganization(ctx, organization)
		next.ServeHTTP(w, r.WithContext(ctx))
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Cloudivision-Organization,X-Cloudivision-Project")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
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
	writeJSONStatus(w, status, ErrorResponse{
		Code: code, Message: redact.MaskString(message), RequestID: w.Header().Get(observability.RequestIDHeader), Violations: apiErr.violations,
	})
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
	status     int
	code       string
	message    string
	violations []policy.Violation
}

func policyDenied(decision policy.Decision) error {
	return apiError{status: http.StatusForbidden, code: "policy_denied", message: decision.Message, violations: decision.Violations}
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

func buildRunFromWebhook(namespace string, repository *cicdv1alpha1.Repository, project *cicdv1alpha1.Project, template *cicdv1alpha1.PipelineTemplate, event webhook.Event) (cicdv1alpha1.BuildRun, error) {
	imageRepository := template.Spec.Build.Image
	if imageRepository == "" {
		imageRepository = strings.TrimRight(project.Spec.DefaultRegistry, "/") + "/" + repository.Name
	}
	name := buildRunNameForWebhook(repository.Name, event.EventID)
	tagTemplate := ""
	if project.Spec.ImageTagPolicy != nil {
		tagTemplate = project.Spec.ImageTagPolicy.DefaultTagTemplate
	}
	tag, _, err := buildlogic.ResolveImageTag("", tagTemplate, buildlogic.TagInput{
		Branch: event.Branch, CommitSHA: event.CommitSHA, Revision: event.CommitSHA,
		BuildRunName: name, Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return cicdv1alpha1.BuildRun{}, err
	}
	return cicdv1alpha1.BuildRun{
		ObjectMeta: objectMeta(name, namespace),
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
	}, nil
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

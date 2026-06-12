package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Server struct {
	Client           client.Client
	LogReader        PodLogReader
	Logger           *slog.Logger
	DefaultNamespace string
	AuthMode         string
	CORSOrigins      []string
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.health)
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
	return s.logging(s.cors(s.auth(mux)))
}

func (s Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
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
		if s.AuthMode == "disabled" {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusNotImplemented, "auth_not_implemented", "authentication is not implemented; set CLOU_DIVISION_AUTH_MODE=disabled for development only")
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
		logger.Info("api request", "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r)
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
	writeJSONStatus(w, status, ErrorResponse{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	writeJSONStatus(w, status, value)
}

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
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

package api

import (
	"encoding/json"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

type ProjectRequest struct {
	Name      string                   `json:"name"`
	Namespace string                   `json:"namespace,omitempty"`
	Spec      cicdv1alpha1.ProjectSpec `json:"spec"`
}

type ProjectResponse struct {
	Name      string                     `json:"name"`
	Namespace string                     `json:"namespace"`
	Spec      cicdv1alpha1.ProjectSpec   `json:"spec"`
	Status    cicdv1alpha1.ProjectStatus `json:"status"`
}

type RepositoryRequest struct {
	Name      string                      `json:"name"`
	Namespace string                      `json:"namespace,omitempty"`
	Spec      cicdv1alpha1.RepositorySpec `json:"spec"`
}

type RepositoryResponse struct {
	Name      string                        `json:"name"`
	Namespace string                        `json:"namespace"`
	Spec      cicdv1alpha1.RepositorySpec   `json:"spec"`
	Status    cicdv1alpha1.RepositoryStatus `json:"status"`
}

type PipelineTemplateRequest struct {
	Name      string                            `json:"name"`
	Namespace string                            `json:"namespace,omitempty"`
	Spec      cicdv1alpha1.PipelineTemplateSpec `json:"spec"`
}

type PipelineTemplateResponse struct {
	Name      string                              `json:"name"`
	Namespace string                              `json:"namespace"`
	Spec      cicdv1alpha1.PipelineTemplateSpec   `json:"spec"`
	Status    cicdv1alpha1.PipelineTemplateStatus `json:"status"`
}

type BuildRunRequest struct {
	Name      string                    `json:"name"`
	Namespace string                    `json:"namespace,omitempty"`
	Spec      cicdv1alpha1.BuildRunSpec `json:"spec"`
}

type BuildRunResponse struct {
	Name      string                      `json:"name"`
	Namespace string                      `json:"namespace"`
	Spec      cicdv1alpha1.BuildRunSpec   `json:"spec"`
	Status    cicdv1alpha1.BuildRunStatus `json:"status"`
}

type EnvironmentResponse struct {
	Name      string                         `json:"name"`
	Namespace string                         `json:"namespace"`
	Spec      cicdv1alpha1.EnvironmentSpec   `json:"spec"`
	Status    cicdv1alpha1.EnvironmentStatus `json:"status"`
}

type ReleaseResponse struct {
	Name      string                     `json:"name"`
	Namespace string                     `json:"namespace"`
	Spec      cicdv1alpha1.ReleaseSpec   `json:"spec"`
	Status    cicdv1alpha1.ReleaseStatus `json:"status"`
}

type ReleaseApprovalRequest struct {
	Actor   string `json:"actor"`
	Comment string `json:"comment,omitempty"`
}

type LogsResponse struct {
	Namespace string   `json:"namespace"`
	BuildRun  string   `json:"buildRun"`
	PodName   string   `json:"podName,omitempty"`
	Lines     []string `json:"lines"`
}

type WebhookResponse struct {
	Repository string           `json:"repository"`
	EventID    string           `json:"eventID"`
	BuildRun   BuildRunResponse `json:"buildRun"`
	Created    bool             `json:"created"`
}

type AuditEventResponse struct {
	ID         string          `json:"id,omitempty"`
	Type       string          `json:"type"`
	Actor      string          `json:"actor,omitempty"`
	Project    string          `json:"project,omitempty"`
	Repository string          `json:"repository,omitempty"`
	BuildRun   string          `json:"buildRun,omitempty"`
	Release    string          `json:"release,omitempty"`
	Message    string          `json:"message,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  metav1.Time     `json:"createdAt,omitempty"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

func projectDTO(project cicdv1alpha1.Project) ProjectResponse {
	return ProjectResponse{Name: project.Name, Namespace: project.Namespace, Spec: project.Spec, Status: project.Status}
}

func repositoryDTO(repository cicdv1alpha1.Repository) RepositoryResponse {
	return RepositoryResponse{Name: repository.Name, Namespace: repository.Namespace, Spec: repository.Spec, Status: repository.Status}
}

func pipelineTemplateDTO(template cicdv1alpha1.PipelineTemplate) PipelineTemplateResponse {
	return PipelineTemplateResponse{Name: template.Name, Namespace: template.Namespace, Spec: template.Spec, Status: template.Status}
}

func buildRunDTO(buildRun cicdv1alpha1.BuildRun) BuildRunResponse {
	return BuildRunResponse{Name: buildRun.Name, Namespace: buildRun.Namespace, Spec: buildRun.Spec, Status: buildRun.Status}
}

func environmentDTO(environment cicdv1alpha1.Environment) EnvironmentResponse {
	return EnvironmentResponse{Name: environment.Name, Namespace: environment.Namespace, Spec: environment.Spec, Status: environment.Status}
}

func releaseDTO(release cicdv1alpha1.Release) ReleaseResponse {
	return ReleaseResponse{Name: release.Name, Namespace: release.Namespace, Spec: release.Spec, Status: release.Status}
}

func auditEventDTO(event audit.Event) AuditEventResponse {
	createdAt := metav1.NewTime(event.CreatedAt)
	return AuditEventResponse{
		ID:         event.ID,
		Type:       event.Type,
		Actor:      event.Actor,
		Project:    event.Project,
		Repository: event.Repository,
		BuildRun:   event.BuildRun,
		Release:    event.Release,
		Message:    event.Message,
		Metadata:   event.Metadata,
		CreatedAt:  createdAt,
	}
}

func objectMeta(name, namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: namespace}
}

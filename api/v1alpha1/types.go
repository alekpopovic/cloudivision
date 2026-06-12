package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

// ProjectStatus defines the observed state of a Project.
type ProjectStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Project represents an isolated CI/CD project or team boundary.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project resources.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// RepositorySpec defines the desired state of a Repository.
type RepositorySpec struct {
	ProjectRef string `json:"projectRef,omitempty"`
	URL        string `json:"url,omitempty"`
	Provider   string `json:"provider,omitempty"`
}

// RepositoryStatus defines the observed state of a Repository.
type RepositoryStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Repository represents source code watched by cloudivision.
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository resources.
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

// PipelineTemplateSpec defines reusable CI pipeline steps.
type PipelineTemplateSpec struct {
	Description string   `json:"description,omitempty"`
	Steps       []string `json:"steps,omitempty"`
}

// PipelineTemplateStatus defines the observed state of a PipelineTemplate.
type PipelineTemplateStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PipelineTemplate defines reusable build pipeline configuration.
type PipelineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineTemplateSpec   `json:"spec,omitempty"`
	Status PipelineTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipelineTemplateList contains a list of PipelineTemplate resources.
type PipelineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineTemplate `json:"items"`
}

// BuildRunSpec defines the desired state of a BuildRun.
type BuildRunSpec struct {
	ProjectRef          string `json:"projectRef,omitempty"`
	RepositoryRef       string `json:"repositoryRef,omitempty"`
	PipelineTemplateRef string `json:"pipelineTemplateRef,omitempty"`
	CommitSHA           string `json:"commitSHA,omitempty"`
	Image               string `json:"image,omitempty"`
}

// BuildRunStatus defines the observed state of a BuildRun.
type BuildRunStatus struct {
	Phase       string             `json:"phase,omitempty"`
	JobName     string             `json:"jobName,omitempty"`
	ArtifactRef string             `json:"artifactRef,omitempty"`
	Conditions  []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BuildRun represents one CI execution.
type BuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildRunSpec   `json:"spec,omitempty"`
	Status BuildRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BuildRunList contains a list of BuildRun resources.
type BuildRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildRun `json:"items"`
}

// EnvironmentSpec defines the desired state of an Environment.
type EnvironmentSpec struct {
	ProjectRef    string `json:"projectRef,omitempty"`
	GitOpsRepoURL string `json:"gitOpsRepoURL,omitempty"`
	ClusterRef    string `json:"clusterRef,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
}

// EnvironmentStatus defines the observed state of an Environment.
type EnvironmentStatus struct {
	Phase      string             `json:"phase,omitempty"`
	SyncStatus string             `json:"syncStatus,omitempty"`
	Health     string             `json:"health,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Environment represents a GitOps-managed deployment target.
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvironmentList contains a list of Environment resources.
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

// ReleaseSpec defines the desired state of a Release.
type ReleaseSpec struct {
	ProjectRef     string `json:"projectRef,omitempty"`
	EnvironmentRef string `json:"environmentRef,omitempty"`
	BuildRunRef    string `json:"buildRunRef,omitempty"`
	ArtifactRef    string `json:"artifactRef,omitempty"`
}

// ReleaseStatus defines the observed state of a Release.
type ReleaseStatus struct {
	Phase      string             `json:"phase,omitempty"`
	SyncStatus string             `json:"syncStatus,omitempty"`
	Health     string             `json:"health,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Release represents a GitOps deployment request for an artifact.
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release resources.
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

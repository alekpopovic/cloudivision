package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectPhase string

const (
	ProjectPhasePending ProjectPhase = "Pending"
	ProjectPhaseReady   ProjectPhase = "Ready"
	ProjectPhaseError   ProjectPhase = "Error"
)

type PodSecurityLevel string

const (
	PodSecurityLevelBaseline   PodSecurityLevel = "baseline"
	PodSecurityLevelRestricted PodSecurityLevel = "restricted"
)

type NetworkPolicyMode string

const (
	NetworkPolicyModeDisabled        NetworkPolicyMode = "disabled"
	NetworkPolicyModeDefaultDeny     NetworkPolicyMode = "defaultDeny"
	NetworkPolicyModeEgressAllowList NetworkPolicyMode = "egressAllowList"
)

type RepositoryPhase string

const (
	RepositoryPhasePending RepositoryPhase = "Pending"
	RepositoryPhaseReady   RepositoryPhase = "Ready"
	RepositoryPhaseError   RepositoryPhase = "Error"
)

type RepositoryProvider string

const (
	RepositoryProviderGitHub  RepositoryProvider = "github"
	RepositoryProviderGitLab  RepositoryProvider = "gitlab"
	RepositoryProviderGitea   RepositoryProvider = "gitea"
	RepositoryProviderGeneric RepositoryProvider = "generic"
)

type PipelineTemplatePhase string

const (
	PipelineTemplatePhaseReady PipelineTemplatePhase = "Ready"
	PipelineTemplatePhaseError PipelineTemplatePhase = "Error"
)

type BuildBuilder string

const (
	BuildBuilderBuildKit BuildBuilder = "buildkit"
	BuildBuilderBuildah  BuildBuilder = "buildah"
	BuildBuilderNone     BuildBuilder = "none"
)

type BuildRunPhase string

const (
	BuildRunPhasePending   BuildRunPhase = "Pending"
	BuildRunPhaseQueued    BuildRunPhase = "Queued"
	BuildRunPhaseRunning   BuildRunPhase = "Running"
	BuildRunPhaseSucceeded BuildRunPhase = "Succeeded"
	BuildRunPhaseFailed    BuildRunPhase = "Failed"
	BuildRunPhaseCancelled BuildRunPhase = "Cancelled"
)

type TriggerType string

const (
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeAPI      TriggerType = "api"
)

type ExecutorType string

const (
	ExecutorTypeJob    ExecutorType = "job"
	ExecutorTypeTekton ExecutorType = "tekton"
)

type GitOpsStrategy string

const (
	GitOpsStrategyHelmValues     GitOpsStrategy = "helm-values"
	GitOpsStrategyKustomizeImage GitOpsStrategy = "kustomize-image"
	GitOpsStrategyRawYAML        GitOpsStrategy = "raw-yaml"
)

type EnvironmentPhase string

const (
	EnvironmentPhasePending EnvironmentPhase = "Pending"
	EnvironmentPhaseReady   EnvironmentPhase = "Ready"
	EnvironmentPhaseError   EnvironmentPhase = "Error"
)

type EnvironmentType string

const (
	EnvironmentTypeDev        EnvironmentType = "dev"
	EnvironmentTypeStaging    EnvironmentType = "staging"
	EnvironmentTypeProduction EnvironmentType = "production"
	EnvironmentTypeCustom     EnvironmentType = "custom"
)

type GitOpsProvider string

const (
	GitOpsProviderArgoCD  GitOpsProvider = "argocd"
	GitOpsProviderFlux    GitOpsProvider = "flux"
	GitOpsProviderGeneric GitOpsProvider = "generic"
)

type ReleasePhase string

const (
	ReleasePhasePending          ReleasePhase = "Pending"
	ReleasePhaseAwaitingApproval ReleasePhase = "AwaitingApproval"
	ReleasePhaseDeploying        ReleasePhase = "Deploying"
	ReleasePhaseDeployed         ReleasePhase = "Deployed"
	ReleasePhaseFailed           ReleasePhase = "Failed"
	ReleasePhaseRolledBack       ReleasePhase = "RolledBack"
)

type ReleaseStrategy string

const (
	ReleaseStrategyGitOps ReleaseStrategy = "gitops"
)

type SecretKeyRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Key string `json:"key,omitempty"`
}

type RequiredSecretKeyRef struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

type ProjectIsolation struct {
	CreateNamespace bool `json:"createNamespace"`
	// +kubebuilder:validation:Enum=baseline;restricted
	PodSecurityLevel PodSecurityLevel `json:"podSecurityLevel"`
	// +kubebuilder:validation:Enum=disabled;defaultDeny;egressAllowList
	NetworkPolicyMode NetworkPolicyMode `json:"networkPolicyMode"`
}

type ProjectSpec struct {
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`
	// +optional
	Description string `json:"description,omitempty"`
	// +kubebuilder:validation:MinLength=1
	OwnerTeam string `json:"ownerTeam"`
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
	// +kubebuilder:validation:MinLength=1
	DefaultRegistry string `json:"defaultRegistry"`
	// +kubebuilder:default:=main
	DefaultBranch string `json:"defaultBranch,omitempty"`
	// +optional
	ServiceAccountName string           `json:"serviceAccountName,omitempty"`
	Isolation          ProjectIsolation `json:"isolation"`
}

type ProjectStatus struct {
	// +kubebuilder:validation:Enum=Pending;Ready;Error
	Phase              ProjectPhase       `json:"phase,omitempty"`
	NamespaceReady     bool               `json:"namespaceReady,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="NamespaceReady",type=boolean,JSONPath=`.status.namespaceReady`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

type RepositoryWebhook struct {
	Enabled   bool                 `json:"enabled"`
	SecretRef RequiredSecretKeyRef `json:"secretRef"`
	Events    []string             `json:"events,omitempty"`
}

type RepositorySpec struct {
	// +kubebuilder:validation:MinLength=1
	ProjectRef string `json:"projectRef"`
	// +kubebuilder:validation:Enum=github;gitlab;gitea;generic
	Provider RepositoryProvider `json:"provider"`
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`
	// +kubebuilder:validation:MinLength=1
	DefaultBranch       string            `json:"defaultBranch"`
	CredentialSecretRef SecretKeyRef      `json:"credentialSecretRef,omitempty"`
	Webhook             RepositoryWebhook `json:"webhook,omitempty"`
	// +kubebuilder:validation:MinLength=1
	PipelineTemplateRef string `json:"pipelineTemplateRef"`
}

type RepositoryStatus struct {
	// +kubebuilder:validation:Enum=Pending;Ready;Error
	Phase RepositoryPhase `json:"phase,omitempty"`
	// +optional
	LastWebhookAt      *metav1.Time       `json:"lastWebhookAt,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

type ParamSpec struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	Description string `json:"description,omitempty"`
	// +optional
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required"`
}

type PipelineStep struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Image   string   `json:"image"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	// +optional
	WorkingDir string          `json:"workingDir,omitempty"`
	Env        []corev1.EnvVar `json:"env,omitempty"`
	// +optional
	TimeoutSeconds  int  `json:"timeoutSeconds,omitempty"`
	ContinueOnError bool `json:"continueOnError,omitempty"`
}

type PipelineBuildSpec struct {
	Enabled bool `json:"enabled"`
	// +kubebuilder:default:=.
	ContextDir string `json:"contextDir,omitempty"`
	// +kubebuilder:default:=Dockerfile
	Dockerfile string `json:"dockerfile,omitempty"`
	// +kubebuilder:validation:Enum=buildkit;buildah;none
	Builder BuildBuilder `json:"builder"`
	// +optional
	Image string `json:"image,omitempty"`
	// +kubebuilder:default:=true
	Push bool `json:"push"`
}

type PipelineResourceSpec struct {
	CPURequest    string `json:"cpuRequest,omitempty"`
	CPULimit      string `json:"cpuLimit,omitempty"`
	MemoryRequest string `json:"memoryRequest,omitempty"`
	MemoryLimit   string `json:"memoryLimit,omitempty"`
	// +kubebuilder:validation:Minimum=1
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

type PipelineSecuritySpec struct {
	// +kubebuilder:default:=false
	AllowPrivileged bool `json:"allowPrivileged,omitempty"`
	// +kubebuilder:default:=true
	RunAsNonRoot bool `json:"runAsNonRoot"`
	// +kubebuilder:default:=false
	ReadOnlyRootFilesystem bool `json:"readOnlyRootFilesystem,omitempty"`
}

type PipelineSupplyChainSpec struct {
	GenerateSBOM            bool `json:"generateSBOM,omitempty"`
	ScanImage               bool `json:"scanImage,omitempty"`
	SignImage               bool `json:"signImage,omitempty"`
	RequireSignedBaseImages bool `json:"requireSignedBaseImages,omitempty"`
}

type PipelineTemplateSpec struct {
	// +optional
	ProjectRef string `json:"projectRef,omitempty"`
	// +optional
	Description string                  `json:"description,omitempty"`
	Params      []ParamSpec             `json:"params,omitempty"`
	Steps       []PipelineStep          `json:"steps,omitempty"`
	Build       PipelineBuildSpec       `json:"build,omitempty"`
	Resources   PipelineResourceSpec    `json:"resources,omitempty"`
	Security    PipelineSecuritySpec    `json:"security,omitempty"`
	SupplyChain PipelineSupplyChainSpec `json:"supplyChain,omitempty"`
}

type PipelineTemplateStatus struct {
	// +kubebuilder:validation:Enum=Ready;Error
	Phase              PipelineTemplatePhase `json:"phase,omitempty"`
	Conditions         []metav1.Condition    `json:"conditions,omitempty"`
	ObservedGeneration int64                 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef`
// +kubebuilder:printcolumn:name="Builder",type=string,JSONPath=`.spec.build.builder`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

type TriggeredBy struct {
	// +kubebuilder:validation:Enum=webhook;manual;schedule;api
	Type TriggerType `json:"type"`
	// +optional
	Actor string `json:"actor,omitempty"`
	// +optional
	EventID string `json:"eventID,omitempty"`
}

type ImageRef struct {
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository"`
	// +optional
	Tag string `json:"tag,omitempty"`
	// +optional
	Digest string `json:"digest,omitempty"`
}

type BuildRunGitOpsSpec struct {
	Enabled bool `json:"enabled"`
	// +optional
	RepoURL string `json:"repoURL,omitempty"`
	// +optional
	Branch string `json:"branch,omitempty"`
	// +optional
	Path string `json:"path,omitempty"`
	// +kubebuilder:validation:Enum=helm-values;kustomize-image;raw-yaml
	Strategy GitOpsStrategy `json:"strategy,omitempty"`
	// +optional
	EnvironmentRef string `json:"environmentRef,omitempty"`
}

type BuildRunSpec struct {
	// +kubebuilder:validation:MinLength=1
	ProjectRef string `json:"projectRef"`
	// +kubebuilder:validation:MinLength=1
	RepositoryRef string `json:"repositoryRef"`
	// +kubebuilder:validation:MinLength=1
	PipelineTemplateRef string `json:"pipelineTemplateRef"`
	// +kubebuilder:validation:MinLength=1
	Revision string `json:"revision"`
	// +optional
	Branch string `json:"branch,omitempty"`
	// +optional
	CommitSHA   string            `json:"commitSHA,omitempty"`
	TriggeredBy TriggeredBy       `json:"triggeredBy"`
	Image       ImageRef          `json:"image"`
	Params      map[string]string `json:"params,omitempty"`
	// +kubebuilder:validation:Enum=job;tekton
	// +kubebuilder:default:=job
	Executor ExecutorType       `json:"executor,omitempty"`
	GitOps   BuildRunGitOpsSpec `json:"gitOps,omitempty"`
}

type ObjectRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type FailureStatus struct {
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type BuildRunLogStatus struct {
	PodName       string `json:"podName,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	// +kubebuilder:validation:MaxItems=20
	LastLines []string `json:"lastLines,omitempty"`
}

type BuildRunSupplyChainStatus struct {
	SBOMPath          string `json:"sbomPath,omitempty"`
	SBOMDigest        string `json:"sbomDigest,omitempty"`
	SignatureRef      string `json:"signatureRef,omitempty"`
	ProvenanceRef     string `json:"provenanceRef,omitempty"`
	ScannerResultsRef string `json:"scannerResultsRef,omitempty"`
}

type BuildRunStatus struct {
	// +kubebuilder:validation:Enum=Pending;Queued;Running;Succeeded;Failed;Cancelled
	Phase              BuildRunPhase      `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	CompletedAt    *metav1.Time              `json:"completedAt,omitempty"`
	JobRef         ObjectRef                 `json:"jobRef,omitempty"`
	PipelineRunRef ObjectRef                 `json:"pipelineRunRef,omitempty"`
	Image          ImageRef                  `json:"image,omitempty"`
	SupplyChain    BuildRunSupplyChainStatus `json:"supplyChain,omitempty"`
	Failure        FailureStatus             `json:"failure,omitempty"`
	Log            BuildRunLogStatus         `json:"log,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef`
// +kubebuilder:printcolumn:name="Revision",type=string,JSONPath=`.spec.revision`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.status.image.repository`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

type EnvironmentGitOpsSpec struct {
	// +kubebuilder:validation:Enum=argocd;flux;generic
	Provider GitOpsProvider `json:"provider"`
	// +optional
	ApplicationName string `json:"applicationName,omitempty"`
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type EnvironmentPolicySpec struct {
	RequireSignedImages          bool `json:"requireSignedImages,omitempty"`
	RequireSBOM                  bool `json:"requireSBOM,omitempty"`
	BlockCriticalVulnerabilities bool `json:"blockCriticalVulnerabilities,omitempty"`
}

type EnvironmentSpec struct {
	// +kubebuilder:validation:MinLength=1
	ProjectRef string `json:"projectRef"`
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
	// +kubebuilder:validation:Enum=dev;staging;production;custom
	Type             EnvironmentType       `json:"type"`
	RequiresApproval bool                  `json:"requiresApproval"`
	GitOps           EnvironmentGitOpsSpec `json:"gitOps"`
	Policy           EnvironmentPolicySpec `json:"policy,omitempty"`
}

type EnvironmentStatus struct {
	// +kubebuilder:validation:Enum=Pending;Ready;Error
	Phase              EnvironmentPhase   `json:"phase,omitempty"`
	SyncStatus         string             `json:"syncStatus,omitempty"`
	HealthStatus       string             `json:"healthStatus,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

type ReleaseApprovalSpec struct {
	Required bool `json:"required"`
	// +optional
	ApprovedBy string `json:"approvedBy,omitempty"`
	// +optional
	ApprovedAt *metav1.Time `json:"approvedAt,omitempty"`
	// +optional
	RejectedBy string `json:"rejectedBy,omitempty"`
	// +optional
	RejectedAt *metav1.Time `json:"rejectedAt,omitempty"`
	// +optional
	Comment string `json:"comment,omitempty"`
}

type ReleaseSpec struct {
	// +kubebuilder:validation:MinLength=1
	ProjectRef string `json:"projectRef"`
	// +optional
	EnvironmentRef string `json:"environmentRef,omitempty"`
	// +kubebuilder:validation:MinLength=1
	BuildRunRef string              `json:"buildRunRef"`
	Image       ImageRef            `json:"image"`
	Approval    ReleaseApprovalSpec `json:"approval,omitempty"`
	// +kubebuilder:validation:Enum=gitops
	Strategy ReleaseStrategy `json:"strategy"`
}

type ReleaseDeploymentStatus struct {
	Provider        string `json:"provider,omitempty"`
	ApplicationName string `json:"applicationName,omitempty"`
	SyncStatus      string `json:"syncStatus,omitempty"`
	HealthStatus    string `json:"healthStatus,omitempty"`
}

type ReleaseStatus struct {
	// +kubebuilder:validation:Enum=Pending;AwaitingApproval;Deploying;Deployed;Failed;RolledBack
	Phase              ReleasePhase       `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// +optional
	GitCommit  string                  `json:"gitCommit,omitempty"`
	Deployment ReleaseDeploymentStatus `json:"deployment,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Environment",type=string,JSONPath=`.spec.environmentRef`
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image.repository`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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

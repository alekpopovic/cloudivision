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

type RegistryProviderType string

const (
	RegistryProviderGeneric RegistryProviderType = "generic"
	RegistryProviderGHCR    RegistryProviderType = "ghcr"
	RegistryProviderGitLab  RegistryProviderType = "gitlab"
	RegistryProviderHarbor  RegistryProviderType = "harbor"
	RegistryProviderECR     RegistryProviderType = "ecr"
	RegistryProviderGCR     RegistryProviderType = "gcr"
	RegistryProviderACR     RegistryProviderType = "acr"
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

type BuildCacheMode string

const (
	BuildCacheModeInline   BuildCacheMode = "inline"
	BuildCacheModeRegistry BuildCacheMode = "registry"
	BuildCacheModeLocal    BuildCacheMode = "local"
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
	ReleasePhasePending               ReleasePhase = "Pending"
	ReleasePhaseAwaitingApproval      ReleasePhase = "AwaitingApproval"
	ReleasePhasePreparingGitOpsChange ReleasePhase = "PreparingGitOpsChange"
	ReleasePhaseGitOpsChangeCommitted ReleasePhase = "GitOpsChangeCommitted"
	ReleasePhaseWaitingForSync        ReleasePhase = "WaitingForSync"
	ReleasePhaseDeployed              ReleasePhase = "Deployed"
	ReleasePhaseRolledBack            ReleasePhase = "RolledBack"
	ReleasePhaseFailedValidation      ReleasePhase = "FailedValidation"
	ReleasePhaseFailedApproval        ReleasePhase = "FailedApproval"
	ReleasePhaseFailedGitClone        ReleasePhase = "FailedGitClone"
	ReleasePhaseFailedGitCommit       ReleasePhase = "FailedGitCommit"
	ReleasePhaseFailedGitPush         ReleasePhase = "FailedGitPush"
	ReleasePhaseFailedProviderStatus  ReleasePhase = "FailedProviderStatus"
	ReleasePhaseTimedOut              ReleasePhase = "TimedOut"
)

type ReleaseStrategy string

const (
	ReleaseStrategyGitOps ReleaseStrategy = "gitops"
)

type PromotionMode string

const (
	PromotionModeDirectCommit PromotionMode = "direct-commit"
	PromotionModePullRequest  PromotionMode = "pull-request"
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

type ProjectRegistrySpec struct {
	// +kubebuilder:validation:Enum=generic;ghcr;gitlab;harbor;ecr;gcr;acr
	Provider RegistryProviderType `json:"provider,omitempty"`
	// +optional
	ImagePrefix string `json:"imagePrefix,omitempty"`
	// +optional
	CredentialSecretRef *SecretKeyRef `json:"credentialSecretRef,omitempty"`
}

type ProjectImageTagPolicySpec struct {
	// +optional
	DefaultTagTemplate string `json:"defaultTagTemplate,omitempty"`
}

type ProjectQuotaSpec struct {
	// +kubebuilder:validation:Minimum=1
	MaxConcurrentBuildRuns int `json:"maxConcurrentBuildRuns,omitempty"`
	// +kubebuilder:validation:Minimum=1
	MaxQueuedBuildRuns int `json:"maxQueuedBuildRuns,omitempty"`
	// +optional
	MaxCPU string `json:"maxCPU,omitempty"`
	// +optional
	MaxMemory string `json:"maxMemory,omitempty"`
	// +kubebuilder:validation:Minimum=1
	MaxBuildDurationSeconds int `json:"maxBuildDurationSeconds,omitempty"`
	// +optional
	MaxArtifactsSize string `json:"maxArtifactsSize,omitempty"`
	// +optional
	MaxLogSize string `json:"maxLogSize,omitempty"`
}

type NotificationFilters struct {
	Project     string `json:"project,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Environment string `json:"environment,omitempty"`
	Phase       string `json:"phase,omitempty"`
}

type ProjectNotificationSpec struct {
	Enabled bool `json:"enabled"`
	// +kubebuilder:validation:Enum=webhook;slack;teams;email
	Provider  string              `json:"provider"`
	SecretRef *SecretKeyRef       `json:"secretRef,omitempty"`
	Events    []string            `json:"events,omitempty"`
	Filters   NotificationFilters `json:"filters,omitempty"`
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
	// +optional
	Registry *ProjectRegistrySpec `json:"registry,omitempty"`
	// +optional
	ImageTagPolicy *ProjectImageTagPolicySpec `json:"imageTagPolicy,omitempty"`
	// +optional
	Quotas *ProjectQuotaSpec `json:"quotas,omitempty"`
	// Notifications routes selected controller events without exposing provider credentials.
	// +optional
	Notifications *ProjectNotificationSpec `json:"notifications,omitempty"`
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

type RepositoryRefFilters struct {
	// +optional
	Include []string `json:"include,omitempty"`
	// +optional
	Exclude []string `json:"exclude,omitempty"`
}

type RepositoryPullRequestFilters struct {
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	Events []string `json:"events,omitempty"`
	// +kubebuilder:default:=false
	BuildForks bool `json:"buildForks,omitempty"`
	// +kubebuilder:default:=false
	RequireTrustedActor bool `json:"requireTrustedActor,omitempty"`
}

type RepositoryWebhook struct {
	Enabled   bool                 `json:"enabled"`
	SecretRef RequiredSecretKeyRef `json:"secretRef,omitempty"`
	Events    []string             `json:"events,omitempty"`
	// +optional
	BranchFilters RepositoryRefFilters `json:"branchFilters,omitempty"`
	// +optional
	TagFilters RepositoryRefFilters `json:"tagFilters,omitempty"`
	// +optional
	PullRequest RepositoryPullRequestFilters `json:"pullRequest,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!has(self.webhook) || !self.webhook.enabled || has(self.webhook.secretRef)",message="webhook.secretRef is required when webhook.enabled is true"
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
	// +optional
	Artifacts *PipelineArtifactSpec `json:"artifacts,omitempty"`
}

type PipelineArtifactSpec struct {
	// +kubebuilder:validation:MinItems=1
	Paths    []string `json:"paths"`
	Optional bool     `json:"optional,omitempty"`
	// +kubebuilder:validation:Minimum=1
	RetentionDays int `json:"retentionDays,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="!self.enabled || self.mode == 'inline' || (has(self.ref) && self.ref != ”)",message="cache.ref is required for registry and local cache modes"
type PipelineBuildCacheSpec struct {
	Enabled bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Enum=inline;registry;local
	// +kubebuilder:default:=inline
	Mode BuildCacheMode `json:"mode,omitempty"`
	// +optional
	Ref string `json:"ref,omitempty"`
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
	// +optional
	BuildArgs map[string]string `json:"buildArgs,omitempty"`
	// +optional
	Target string `json:"target,omitempty"`
	// +kubebuilder:validation:MaxItems=16
	// +listType=set
	Platforms []string `json:"platforms,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	Cache PipelineBuildCacheSpec `json:"cache,omitempty"`
}

type DependencyCacheMode string

const (
	DependencyCacheModePVC           DependencyCacheMode = "pvc"
	DependencyCacheModeRegistry      DependencyCacheMode = "registry"
	DependencyCacheModeObjectStorage DependencyCacheMode = "object-storage"
)

// PipelineCacheSpec configures an opt-in dependency cache. Cache entries are
// always scoped by the BuildRun project and repository before Key is applied.
type PipelineCacheSpec struct {
	Enabled bool `json:"enabled,omitempty"`
	// +kubebuilder:validation:Enum=pvc;registry;object-storage
	Mode DependencyCacheMode `json:"mode,omitempty"`
	// +optional
	Key string `json:"key,omitempty"`
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	Paths []string `json:"paths,omitempty"`
	// +kubebuilder:validation:MaxItems=16
	// +listType=set
	RestoreKeys []string `json:"restoreKeys,omitempty"`
	// +kubebuilder:validation:Minimum=1
	TTLSeconds int `json:"ttlSeconds,omitempty"`
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

// +kubebuilder:validation:XValidation:rule="!(self.cosignKeyless && has(self.signingKeySecretRef))",message="cosign keyless and key-based modes are mutually exclusive"
// +kubebuilder:validation:XValidation:rule="self.signerAdapter != 'cosign' || self.cosignKeyless || has(self.signingKeySecretRef)",message="cosign requires explicit keyless mode or signingKeySecretRef"
type PipelineSupplyChainSpec struct {
	GenerateSBOM            bool `json:"generateSBOM,omitempty"`
	ScanImage               bool `json:"scanImage,omitempty"`
	SignImage               bool `json:"signImage,omitempty"`
	RequireSignedBaseImages bool `json:"requireSignedBaseImages,omitempty"`
	// +kubebuilder:validation:Enum=noop;syft
	// +kubebuilder:default:=noop
	SBOMAdapter string `json:"sbomAdapter,omitempty"`
	// +kubebuilder:validation:Enum=noop;grype
	// +kubebuilder:default:=noop
	ScannerAdapter string `json:"scannerAdapter,omitempty"`
	// +kubebuilder:validation:Enum=noop;cosign
	// +kubebuilder:default:=noop
	SignerAdapter string `json:"signerAdapter,omitempty"`
	// +kubebuilder:validation:Enum=noop;json
	// +kubebuilder:default:=noop
	ProvenanceAdapter string `json:"provenanceAdapter,omitempty"`
	CosignKeyless     bool   `json:"cosignKeyless,omitempty"`
	// +optional
	SigningKeySecretRef *RequiredSecretKeyRef `json:"signingKeySecretRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.steps) && size(self.steps) > 0) || (has(self.build) && self.build.enabled)",message="at least one step or an enabled image build is required"
// +kubebuilder:validation:XValidation:rule="!has(self.cache) || !self.cache.enabled || self.cache.mode != 'registry' || (self.build.enabled && self.build.builder == 'buildkit')",message="registry dependency cache requires an enabled BuildKit image build"
type PipelineTemplateSpec struct {
	// +optional
	ProjectRef string `json:"projectRef,omitempty"`
	// +optional
	Description string      `json:"description,omitempty"`
	Params      []ParamSpec `json:"params,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=64
	// +listType=map
	// +listMapKey=name
	Steps       []PipelineStep          `json:"steps,omitempty"`
	Build       PipelineBuildSpec       `json:"build,omitempty"`
	Cache       PipelineCacheSpec       `json:"cache,omitempty"`
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

// +kubebuilder:validation:XValidation:rule="!self.enabled || has(self.strategy)",message="gitOps.strategy is required when gitOps.enabled is true"
// +kubebuilder:validation:XValidation:rule="!self.enabled || has(self.environmentRef)",message="gitOps.environmentRef is required when gitOps.enabled is true"
// +kubebuilder:validation:XValidation:rule="!self.enabled || has(self.repoURL)",message="gitOps.repoURL is required when gitOps.enabled is true"
type BuildRunGitOpsSpec struct {
	Enabled bool `json:"enabled"`
	// +kubebuilder:validation:MinLength=1
	RepoURL string `json:"repoURL,omitempty"`
	// +optional
	Branch string `json:"branch,omitempty"`
	// +optional
	Path string `json:"path,omitempty"`
	// +kubebuilder:default:=values.yaml
	ValuesFile string `json:"valuesFile,omitempty"`
	// +kubebuilder:default:=image.repository
	ImageRepositoryField string `json:"imageRepositoryField,omitempty"`
	// +kubebuilder:default:=image.tag
	ImageTagField string `json:"imageTagField,omitempty"`
	// +kubebuilder:default:=image.digest
	ImageDigestField string              `json:"imageDigestField,omitempty"`
	Kustomize        GitOpsKustomizeSpec `json:"kustomize,omitempty"`
	RawYAML          GitOpsRawYAMLSpec   `json:"rawYaml,omitempty"`
	// +kubebuilder:validation:Enum=helm-values;kustomize-image;raw-yaml
	Strategy GitOpsStrategy `json:"strategy,omitempty"`
	// +kubebuilder:validation:MinLength=1
	EnvironmentRef string `json:"environmentRef,omitempty"`
}

type GitOpsKustomizeSpec struct {
	// +kubebuilder:default:=kustomization.yaml
	KustomizationFile string `json:"kustomizationFile,omitempty"`
	// +optional
	ImageName string `json:"imageName,omitempty"`
}

type GitOpsRawYAMLSpec struct {
	// +kubebuilder:validation:MaxItems=32
	// +listType=set
	Files []string `json:"files,omitempty"`
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet;CronJob
	WorkloadKind string `json:"workloadKind,omitempty"`
	// +optional
	WorkloadName string `json:"workloadName,omitempty"`
	// +optional
	ContainerName string `json:"containerName,omitempty"`
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
	Backend       string `json:"backend,omitempty"`
	Ref           string `json:"ref,omitempty"`
	PodName       string `json:"podName,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	// +kubebuilder:validation:MaxItems=20
	LastLines []string `json:"lastLines,omitempty"`
}

type BuildRunSupplyChainStatus struct {
	SBOMPath                string `json:"sbomPath,omitempty"`
	SBOMDigest              string `json:"sbomDigest,omitempty"`
	SignatureRef            string `json:"signatureRef,omitempty"`
	ProvenanceRef           string `json:"provenanceRef,omitempty"`
	ScannerResultsRef       string `json:"scannerResultsRef,omitempty"`
	CriticalVulnerabilities int    `json:"criticalVulnerabilities,omitempty"`
	HighVulnerabilities     int    `json:"highVulnerabilities,omitempty"`
	MediumVulnerabilities   int    `json:"mediumVulnerabilities,omitempty"`
	LowVulnerabilities      int    `json:"lowVulnerabilities,omitempty"`
}

type BuildRunArtifactStatus struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Type   string `json:"type,omitempty"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
	Ref    string `json:"ref"`
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`
}

type PolicyViolationStatus struct {
	Policy    string `json:"policy"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	FieldPath string `json:"fieldPath,omitempty"`
}

type PolicyDecisionStatus struct {
	Allowed     bool                    `json:"allowed"`
	Reason      string                  `json:"reason,omitempty"`
	Message     string                  `json:"message,omitempty"`
	EvaluatedAt *metav1.Time            `json:"evaluatedAt,omitempty"`
	Violations  []PolicyViolationStatus `json:"violations,omitempty"`
}

type BuildRunStatus struct {
	// +kubebuilder:validation:Enum=Pending;Queued;Running;Succeeded;Failed;Cancelled
	Phase              BuildRunPhase      `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	CompletedAt    *metav1.Time `json:"completedAt,omitempty"`
	JobRef         ObjectRef    `json:"jobRef,omitempty"`
	PipelineRunRef ObjectRef    `json:"pipelineRunRef,omitempty"`
	// +optional
	Image       *ImageRef                 `json:"image,omitempty"`
	SupplyChain BuildRunSupplyChainStatus `json:"supplyChain,omitempty"`
	Policy      PolicyDecisionStatus      `json:"policy,omitempty"`
	Failure     FailureStatus             `json:"failure,omitempty"`
	Log         BuildRunLogStatus         `json:"log,omitempty"`
	Artifacts   []BuildRunArtifactStatus  `json:"artifacts,omitempty"`
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
	// ResourceKind selects the Flux Kustomization or HelmRelease reader. It is ignored by Argo CD.
	// +optional
	// +kubebuilder:validation:Enum=Kustomization;HelmRelease
	ResourceKind string `json:"resourceKind,omitempty"`
}

type EnvironmentPolicySpec struct {
	RequireImageDigest           bool `json:"requireImageDigest,omitempty"`
	AllowLatest                  bool `json:"allowLatest,omitempty"`
	RequireSignedImages          bool `json:"requireSignedImages,omitempty"`
	RequireSBOM                  bool `json:"requireSBOM,omitempty"`
	BlockCriticalVulnerabilities bool `json:"blockCriticalVulnerabilities,omitempty"`
	FailOnDegraded               bool `json:"failOnDegraded,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="self.type != 'production' || self.requiresApproval",message="production environments must require approval"
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

type PullRequestSpec struct {
	TitleTemplate string   `json:"titleTemplate,omitempty"`
	BodyTemplate  string   `json:"bodyTemplate,omitempty"`
	TargetBranch  string   `json:"targetBranch,omitempty"`
	Reviewers     []string `json:"reviewers,omitempty"`
	Labels        []string `json:"labels,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.image.tag) && size(self.image.tag) > 0) || (has(self.image.digest) && size(self.image.digest) > 0)",message="release image must include a tag or digest"
type ReleaseSpec struct {
	// +kubebuilder:validation:MinLength=1
	ProjectRef string `json:"projectRef"`
	// +kubebuilder:validation:MinLength=1
	EnvironmentRef string `json:"environmentRef"`
	// +kubebuilder:validation:MinLength=1
	BuildRunRef string              `json:"buildRunRef"`
	Image       ImageRef            `json:"image"`
	Approval    ReleaseApprovalSpec `json:"approval,omitempty"`
	// +kubebuilder:validation:Enum=gitops
	Strategy ReleaseStrategy `json:"strategy"`
	// +kubebuilder:validation:Enum=direct-commit;pull-request
	// +kubebuilder:default:=direct-commit
	PromotionMode PromotionMode   `json:"promotionMode,omitempty"`
	PullRequest   PullRequestSpec `json:"pullRequest,omitempty"`
	// DeploymentTimeout limits time spent preparing and waiting for GitOps deployment.
	// +kubebuilder:default:="30m"
	DeploymentTimeout metav1.Duration `json:"deploymentTimeout,omitempty"`
	// PromotedFrom identifies the source Release for an environment promotion.
	PromotedFrom string `json:"promotedFrom,omitempty"`
	// RollbackOf identifies the Release whose deployment is being replaced.
	RollbackOf string `json:"rollbackOf,omitempty"`
	// RollbackTo identifies the previous successful Release whose image is restored.
	RollbackTo string `json:"rollbackTo,omitempty"`
}

type ReleaseDeploymentStatus struct {
	Provider         string `json:"provider,omitempty"`
	ApplicationName  string `json:"applicationName,omitempty"`
	SyncStatus       string `json:"syncStatus,omitempty"`
	HealthStatus     string `json:"healthStatus,omitempty"`
	OperationPhase   string `json:"operationPhase,omitempty"`
	ObservedRevision string `json:"observedRevision,omitempty"`
	ObservedAt       string `json:"observedAt,omitempty"`
}

type ReleaseApprovalStatus struct {
	ApprovedBy string       `json:"approvedBy,omitempty"`
	ApprovedAt *metav1.Time `json:"approvedAt,omitempty"`
	RejectedBy string       `json:"rejectedBy,omitempty"`
	RejectedAt *metav1.Time `json:"rejectedAt,omitempty"`
}

type PullRequestStatus struct {
	Provider     string `json:"provider,omitempty"`
	URL          string `json:"url,omitempty"`
	Reference    string `json:"reference,omitempty"`
	HeadBranch   string `json:"headBranch,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
	MergeStatus  string `json:"mergeStatus,omitempty"`
}

type ReleaseStatus struct {
	// +kubebuilder:validation:Enum=Pending;AwaitingApproval;PreparingGitOpsChange;GitOpsChangeCommitted;WaitingForSync;Deployed;RolledBack;FailedValidation;FailedApproval;FailedGitClone;FailedGitCommit;FailedGitPush;FailedProviderStatus;TimedOut
	Phase              ReleasePhase       `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// +optional
	GitCommit   string                  `json:"gitCommit,omitempty"`
	Deployment  ReleaseDeploymentStatus `json:"deployment,omitempty"`
	Approval    ReleaseApprovalStatus   `json:"approval,omitempty"`
	PullRequest PullRequestStatus       `json:"pullRequest,omitempty"`
	Policy      PolicyDecisionStatus    `json:"policy,omitempty"`
	Failure     FailureStatus           `json:"failure,omitempty"`
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

type ClusterTargetType string

const (
	ClusterTargetTypeBuild  ClusterTargetType = "build"
	ClusterTargetTypeDeploy ClusterTargetType = "deploy"
	ClusterTargetTypeBoth   ClusterTargetType = "both"
)

type ClusterTargetPhase string

const (
	ClusterTargetPhasePending     ClusterTargetPhase = "Pending"
	ClusterTargetPhaseReady       ClusterTargetPhase = "Ready"
	ClusterTargetPhaseUnreachable ClusterTargetPhase = "Unreachable"
)

type ClusterTargetSpec struct {
	DisplayName         string        `json:"displayName"`
	KubeconfigSecretRef *SecretKeyRef `json:"kubeconfigSecretRef,omitempty"`
	Context             string        `json:"context,omitempty"`
	// +kubebuilder:validation:Enum=build;deploy;both
	Type             ClusterTargetType `json:"type"`
	DefaultNamespace string            `json:"defaultNamespace,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
}

type ClusterTargetStatus struct {
	// +kubebuilder:validation:Enum=Pending;Ready;Unreachable
	Phase              ClusterTargetPhase `json:"phase,omitempty"`
	Version            string             `json:"version,omitempty"`
	Reachable          bool               `json:"reachable"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Reachable",type=boolean,JSONPath=`.status.reachable`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterTarget describes an experimental external cluster connection. It is
// health-checked only; schedulers must not place builds on it yet.
type ClusterTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterTargetSpec   `json:"spec,omitempty"`
	Status            ClusterTargetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ClusterTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterTarget `json:"items"`
}

package v1beta1

import (
	"encoding/json"

	alpha "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Reused value types retain their alpha wire representation while the beta
// roots introduce clearer grouping and names.
type (
	ProjectIsolation          = alpha.ProjectIsolation
	ProjectRegistrySpec       = alpha.ProjectRegistrySpec
	ProjectImageTagPolicySpec = alpha.ProjectImageTagPolicySpec
	ProjectQuotaSpec          = alpha.ProjectQuotaSpec
	ProjectNotificationSpec   = alpha.ProjectNotificationSpec
	ProjectStatus             = alpha.ProjectStatus
	SecretKeyRef              = alpha.SecretKeyRef
	RequiredSecretKeyRef      = alpha.RequiredSecretKeyRef
	RepositoryProvider        = alpha.RepositoryProvider
	RepositoryWebhook         = alpha.RepositoryWebhook
	RepositoryStatus          = alpha.RepositoryStatus
	ParamSpec                 = alpha.ParamSpec
	PipelineStep              = alpha.PipelineStep
	PipelineBuildSpec         = alpha.PipelineBuildSpec
	PipelineCacheSpec         = alpha.PipelineCacheSpec
	PipelineResourceSpec      = alpha.PipelineResourceSpec
	PipelineSecuritySpec      = alpha.PipelineSecuritySpec
	PipelineSupplyChainSpec   = alpha.PipelineSupplyChainSpec
	PipelineTemplateStatus    = alpha.PipelineTemplateStatus
	TriggeredBy               = alpha.TriggeredBy
	ImageRef                  = alpha.ImageRef
	ExecutorType              = alpha.ExecutorType
	BuildRunGitOpsSpec        = alpha.BuildRunGitOpsSpec
	BuildRunStatus            = alpha.BuildRunStatus
	EnvironmentType           = alpha.EnvironmentType
	EnvironmentGitOpsSpec     = alpha.EnvironmentGitOpsSpec
	EnvironmentPolicySpec     = alpha.EnvironmentPolicySpec
	EnvironmentStatus         = alpha.EnvironmentStatus
	ReleaseStrategy           = alpha.ReleaseStrategy
	PromotionMode             = alpha.PromotionMode
	PullRequestSpec           = alpha.PullRequestSpec
	ReleaseStatus             = alpha.ReleaseStatus
)

const ExecutorTypeJob = alpha.ExecutorTypeJob

type ProjectSpec struct {
	OrganizationRef    string                     `json:"organizationRef,omitempty"`
	DisplayName        string                     `json:"displayName"`
	Description        string                     `json:"description,omitempty"`
	OwnerTeam          string                     `json:"ownerTeam"`
	WorkloadNamespace  string                     `json:"workloadNamespace"`
	DefaultRegistry    string                     `json:"defaultRegistry"`
	DefaultBranch      string                     `json:"defaultBranch,omitempty"`
	ServiceAccountName string                     `json:"serviceAccountName,omitempty"`
	Isolation          ProjectIsolation           `json:"isolation"`
	Registry           *ProjectRegistrySpec       `json:"registry,omitempty"`
	ImageTagPolicy     *ProjectImageTagPolicySpec `json:"imageTagPolicy,omitempty"`
	Quotas             *ProjectQuotaSpec          `json:"quotas,omitempty"`
	Notifications      *ProjectNotificationSpec   `json:"notifications,omitempty"`
}

type RepositorySpec struct {
	ProjectRef          string             `json:"projectRef"`
	Provider            RepositoryProvider `json:"provider"`
	URL                 string             `json:"url"`
	DefaultBranch       string             `json:"defaultBranch"`
	CredentialSecretRef *SecretKeyRef      `json:"credentialSecretRef,omitempty"`
	Webhook             *RepositoryWebhook `json:"webhook,omitempty"`
	PipelineTemplateRef string             `json:"pipelineTemplateRef"`
}

type PipelineExecutionSpec struct {
	Params []ParamSpec       `json:"params,omitempty"`
	Steps  []PipelineStep    `json:"steps,omitempty"`
	Cache  PipelineCacheSpec `json:"cache,omitempty"`
}

type PipelineTemplateSpec struct {
	ProjectRef     string                  `json:"projectRef,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Execution      PipelineExecutionSpec   `json:"execution,omitempty"`
	ImageBuild     PipelineBuildSpec       `json:"imageBuild,omitempty"`
	Resources      PipelineResourceSpec    `json:"resources,omitempty"`
	SecurityPolicy PipelineSecuritySpec    `json:"securityPolicy,omitempty"`
	SupplyChain    PipelineSupplyChainSpec `json:"supplyChain,omitempty"`
}

type SourceRevision struct {
	Requested string `json:"requested"`
	Resolved  string `json:"resolved,omitempty"`
	Branch    string `json:"branch,omitempty"`
}

type ParameterValue struct {
	String string `json:"string"`
}

type BuildRunSpec struct {
	ProjectRef          string                    `json:"projectRef"`
	RepositoryRef       string                    `json:"repositoryRef"`
	PipelineTemplateRef string                    `json:"pipelineTemplateRef"`
	Source              SourceRevision            `json:"source"`
	TriggeredBy         TriggeredBy               `json:"triggeredBy"`
	ArtifactOutput      ImageRef                  `json:"artifactOutput"`
	Parameters          map[string]ParameterValue `json:"parameters,omitempty"`
	Executor            ExecutorType              `json:"executor,omitempty"`
	Promotion           BuildRunGitOpsSpec        `json:"promotion,omitempty"`
}

type EnvironmentSpec struct {
	ProjectRef       string                `json:"projectRef"`
	DisplayName      string                `json:"displayName"`
	TargetNamespace  string                `json:"targetNamespace"`
	Type             EnvironmentType       `json:"type"`
	RequiresApproval bool                  `json:"requiresApproval"`
	Provider         EnvironmentGitOpsSpec `json:"provider"`
	Policy           EnvironmentPolicySpec `json:"policy,omitempty"`
}

type ApprovalPolicy struct {
	Required bool `json:"required"`
}

// LegacyApprovalDecision exists only so alpha objects round-trip without
// losing mutable approval data. New beta clients must use approval actions.
type LegacyApprovalDecision struct {
	ApprovedBy string       `json:"approvedBy,omitempty"`
	ApprovedAt *metav1.Time `json:"approvedAt,omitempty"`
	RejectedBy string       `json:"rejectedBy,omitempty"`
	RejectedAt *metav1.Time `json:"rejectedAt,omitempty"`
	Comment    string       `json:"comment,omitempty"`
}

type ReleaseSpec struct {
	ProjectRef             string                  `json:"projectRef"`
	EnvironmentRef         string                  `json:"environmentRef"`
	BuildRunRef            string                  `json:"buildRunRef"`
	Artifact               ImageRef                `json:"artifact"`
	ApprovalPolicy         ApprovalPolicy          `json:"approvalPolicy,omitempty"`
	LegacyApprovalDecision *LegacyApprovalDecision `json:"legacyApprovalDecision,omitempty"`
	Strategy               ReleaseStrategy         `json:"strategy"`
	PromotionMode          PromotionMode           `json:"promotionMode,omitempty"`
	PullRequest            PullRequestSpec         `json:"pullRequest,omitempty"`
	DeploymentTimeout      metav1.Duration         `json:"deploymentTimeout,omitempty"`
	PromotedFrom           string                  `json:"promotedFrom,omitempty"`
	RollbackOf             string                  `json:"rollbackOf,omitempty"`
	RollbackTo             string                  `json:"rollbackTo,omitempty"`
}

type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ProjectSpec   `json:"spec,omitempty"`
	Status            ProjectStatus `json:"status,omitempty"`
}
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositorySpec   `json:"spec,omitempty"`
	Status            RepositoryStatus `json:"status,omitempty"`
}
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}
type PipelineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PipelineTemplateSpec   `json:"spec,omitempty"`
	Status            PipelineTemplateStatus `json:"status,omitempty"`
}
type PipelineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineTemplate `json:"items"`
}
type BuildRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BuildRunSpec   `json:"spec,omitempty"`
	Status            BuildRunStatus `json:"status,omitempty"`
}
type BuildRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuildRun `json:"items"`
}
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EnvironmentSpec   `json:"spec,omitempty"`
	Status            EnvironmentStatus `json:"status,omitempty"`
}
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ReleaseSpec   `json:"spec,omitempty"`
	Status            ReleaseStatus `json:"status,omitempty"`
}
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

func deepCopy[T any](in *T) *T {
	data, _ := json.Marshal(in)
	var out T
	_ = json.Unmarshal(data, &out)
	return &out
}
func (in *Project) DeepCopyObject() runtime.Object              { return deepCopy(in) }
func (in *ProjectList) DeepCopyObject() runtime.Object          { return deepCopy(in) }
func (in *Repository) DeepCopyObject() runtime.Object           { return deepCopy(in) }
func (in *RepositoryList) DeepCopyObject() runtime.Object       { return deepCopy(in) }
func (in *PipelineTemplate) DeepCopyObject() runtime.Object     { return deepCopy(in) }
func (in *PipelineTemplateList) DeepCopyObject() runtime.Object { return deepCopy(in) }
func (in *BuildRun) DeepCopyObject() runtime.Object             { return deepCopy(in) }
func (in *BuildRunList) DeepCopyObject() runtime.Object         { return deepCopy(in) }
func (in *Environment) DeepCopyObject() runtime.Object          { return deepCopy(in) }
func (in *EnvironmentList) DeepCopyObject() runtime.Object      { return deepCopy(in) }
func (in *Release) DeepCopyObject() runtime.Object              { return deepCopy(in) }
func (in *ReleaseList) DeepCopyObject() runtime.Object          { return deepCopy(in) }

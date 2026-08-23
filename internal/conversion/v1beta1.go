package conversion

import (
	"encoding/json"
	"fmt"

	alpha "github.com/cloudivision/cloudivision/api/v1alpha1"
	beta "github.com/cloudivision/cloudivision/api/v1beta1"
)

func copyJSON[To any, From any](in From) (To, error) {
	var out To
	data, err := json.Marshal(in)
	if err != nil {
		return out, fmt.Errorf("marshal conversion value: %w", err)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("unmarshal conversion value: %w", err)
	}
	return out, nil
}

func ProjectToV1Beta1(in *alpha.Project) (*beta.Project, error) {
	out := &beta.Project{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = beta.GroupVersion.String()
	out.Spec = beta.ProjectSpec{OrganizationRef: in.Spec.OrganizationRef, DisplayName: in.Spec.DisplayName, Description: in.Spec.Description, OwnerTeam: in.Spec.OwnerTeam, WorkloadNamespace: in.Spec.Namespace, DefaultRegistry: in.Spec.DefaultRegistry, DefaultBranch: in.Spec.DefaultBranch, ServiceAccountName: in.Spec.ServiceAccountName, Isolation: in.Spec.Isolation}
	var err error
	if out.Spec.Registry, err = copyJSON[*beta.ProjectRegistrySpec](in.Spec.Registry); err != nil {
		return nil, err
	}
	if out.Spec.ImageTagPolicy, err = copyJSON[*beta.ProjectImageTagPolicySpec](in.Spec.ImageTagPolicy); err != nil {
		return nil, err
	}
	if out.Spec.Quotas, err = copyJSON[*beta.ProjectQuotaSpec](in.Spec.Quotas); err != nil {
		return nil, err
	}
	if out.Spec.Notifications, err = copyJSON[*beta.ProjectNotificationSpec](in.Spec.Notifications); err != nil {
		return nil, err
	}
	if out.Status, err = copyJSON[beta.ProjectStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func ProjectToV1Alpha1(in *beta.Project) (*alpha.Project, error) {
	out := &alpha.Project{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = alpha.GroupVersion.String()
	out.Spec = alpha.ProjectSpec{OrganizationRef: in.Spec.OrganizationRef, DisplayName: in.Spec.DisplayName, Description: in.Spec.Description, OwnerTeam: in.Spec.OwnerTeam, Namespace: in.Spec.WorkloadNamespace, DefaultRegistry: in.Spec.DefaultRegistry, DefaultBranch: in.Spec.DefaultBranch, ServiceAccountName: in.Spec.ServiceAccountName, Isolation: in.Spec.Isolation}
	var err error
	if out.Spec.Registry, err = copyJSON[*alpha.ProjectRegistrySpec](in.Spec.Registry); err != nil {
		return nil, err
	}
	if out.Spec.ImageTagPolicy, err = copyJSON[*alpha.ProjectImageTagPolicySpec](in.Spec.ImageTagPolicy); err != nil {
		return nil, err
	}
	if out.Spec.Quotas, err = copyJSON[*alpha.ProjectQuotaSpec](in.Spec.Quotas); err != nil {
		return nil, err
	}
	if out.Spec.Notifications, err = copyJSON[*alpha.ProjectNotificationSpec](in.Spec.Notifications); err != nil {
		return nil, err
	}
	if out.Status, err = copyJSON[alpha.ProjectStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func RepositoryToV1Beta1(in *alpha.Repository) (*beta.Repository, error) {
	out := &beta.Repository{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = beta.GroupVersion.String()
	out.Spec = beta.RepositorySpec{ProjectRef: in.Spec.ProjectRef, Provider: in.Spec.Provider, URL: in.Spec.URL, DefaultBranch: in.Spec.DefaultBranch, PipelineTemplateRef: in.Spec.PipelineTemplateRef}
	if in.Spec.CredentialSecretRef.Name != "" || in.Spec.CredentialSecretRef.Key != "" {
		value := in.Spec.CredentialSecretRef
		out.Spec.CredentialSecretRef = &value
	}
	if in.Spec.Webhook.Enabled || in.Spec.Webhook.SecretRef.Name != "" || len(in.Spec.Webhook.Events) > 0 {
		value, err := copyJSON[beta.RepositoryWebhook](in.Spec.Webhook)
		if err != nil {
			return nil, err
		}
		out.Spec.Webhook = &value
	}
	var err error
	if out.Status, err = copyJSON[beta.RepositoryStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func RepositoryToV1Alpha1(in *beta.Repository) (*alpha.Repository, error) {
	out := &alpha.Repository{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = alpha.GroupVersion.String()
	out.Spec = alpha.RepositorySpec{ProjectRef: in.Spec.ProjectRef, Provider: in.Spec.Provider, URL: in.Spec.URL, DefaultBranch: in.Spec.DefaultBranch, PipelineTemplateRef: in.Spec.PipelineTemplateRef}
	if in.Spec.CredentialSecretRef != nil {
		out.Spec.CredentialSecretRef = *in.Spec.CredentialSecretRef
	}
	if in.Spec.Webhook != nil {
		value, err := copyJSON[alpha.RepositoryWebhook](*in.Spec.Webhook)
		if err != nil {
			return nil, err
		}
		out.Spec.Webhook = value
	}
	var err error
	if out.Status, err = copyJSON[alpha.RepositoryStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func PipelineTemplateToV1Beta1(in *alpha.PipelineTemplate) (*beta.PipelineTemplate, error) {
	out := &beta.PipelineTemplate{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = beta.GroupVersion.String()
	out.Spec.ProjectRef, out.Spec.Description = in.Spec.ProjectRef, in.Spec.Description
	var err error
	if out.Spec.Execution.Params, err = copyJSON[[]beta.ParamSpec](in.Spec.Params); err != nil {
		return nil, err
	}
	if out.Spec.Execution.Steps, err = copyJSON[[]beta.PipelineStep](in.Spec.Steps); err != nil {
		return nil, err
	}
	if out.Spec.Execution.Cache, err = copyJSON[beta.PipelineCacheSpec](in.Spec.Cache); err != nil {
		return nil, err
	}
	if out.Spec.ImageBuild, err = copyJSON[beta.PipelineBuildSpec](in.Spec.Build); err != nil {
		return nil, err
	}
	if out.Spec.Resources, err = copyJSON[beta.PipelineResourceSpec](in.Spec.Resources); err != nil {
		return nil, err
	}
	if out.Spec.SecurityPolicy, err = copyJSON[beta.PipelineSecuritySpec](in.Spec.Security); err != nil {
		return nil, err
	}
	if out.Spec.SupplyChain, err = copyJSON[beta.PipelineSupplyChainSpec](in.Spec.SupplyChain); err != nil {
		return nil, err
	}
	if out.Status, err = copyJSON[beta.PipelineTemplateStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func PipelineTemplateToV1Alpha1(in *beta.PipelineTemplate) (*alpha.PipelineTemplate, error) {
	out := &alpha.PipelineTemplate{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = alpha.GroupVersion.String()
	out.Spec.ProjectRef, out.Spec.Description = in.Spec.ProjectRef, in.Spec.Description
	var err error
	if out.Spec.Params, err = copyJSON[[]alpha.ParamSpec](in.Spec.Execution.Params); err != nil {
		return nil, err
	}
	if out.Spec.Steps, err = copyJSON[[]alpha.PipelineStep](in.Spec.Execution.Steps); err != nil {
		return nil, err
	}
	if out.Spec.Cache, err = copyJSON[alpha.PipelineCacheSpec](in.Spec.Execution.Cache); err != nil {
		return nil, err
	}
	if out.Spec.Build, err = copyJSON[alpha.PipelineBuildSpec](in.Spec.ImageBuild); err != nil {
		return nil, err
	}
	if out.Spec.Resources, err = copyJSON[alpha.PipelineResourceSpec](in.Spec.Resources); err != nil {
		return nil, err
	}
	if out.Spec.Security, err = copyJSON[alpha.PipelineSecuritySpec](in.Spec.SecurityPolicy); err != nil {
		return nil, err
	}
	if out.Spec.SupplyChain, err = copyJSON[alpha.PipelineSupplyChainSpec](in.Spec.SupplyChain); err != nil {
		return nil, err
	}
	if out.Status, err = copyJSON[alpha.PipelineTemplateStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func BuildRunToV1Beta1(in *alpha.BuildRun) (*beta.BuildRun, error) {
	out := &beta.BuildRun{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = beta.GroupVersion.String()
	out.Spec = beta.BuildRunSpec{ProjectRef: in.Spec.ProjectRef, RepositoryRef: in.Spec.RepositoryRef, PipelineTemplateRef: in.Spec.PipelineTemplateRef, Source: beta.SourceRevision{Requested: in.Spec.Revision, Resolved: in.Spec.CommitSHA, Branch: in.Spec.Branch}, TriggeredBy: in.Spec.TriggeredBy, ArtifactOutput: in.Spec.Image, Executor: in.Spec.Executor, Promotion: in.Spec.GitOps}
	out.Spec.Parameters = make(map[string]beta.ParameterValue, len(in.Spec.Params))
	for key, value := range in.Spec.Params {
		out.Spec.Parameters[key] = beta.ParameterValue{String: value}
	}
	var err error
	if out.Status, err = copyJSON[beta.BuildRunStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func BuildRunToV1Alpha1(in *beta.BuildRun) (*alpha.BuildRun, error) {
	out := &alpha.BuildRun{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy()}
	out.TypeMeta.APIVersion = alpha.GroupVersion.String()
	out.Spec = alpha.BuildRunSpec{ProjectRef: in.Spec.ProjectRef, RepositoryRef: in.Spec.RepositoryRef, PipelineTemplateRef: in.Spec.PipelineTemplateRef, Revision: in.Spec.Source.Requested, CommitSHA: in.Spec.Source.Resolved, Branch: in.Spec.Source.Branch, TriggeredBy: in.Spec.TriggeredBy, Image: in.Spec.ArtifactOutput, Executor: in.Spec.Executor, GitOps: in.Spec.Promotion}
	out.Spec.Params = make(map[string]string, len(in.Spec.Parameters))
	for key, value := range in.Spec.Parameters {
		out.Spec.Params[key] = value.String
	}
	var err error
	if out.Status, err = copyJSON[alpha.BuildRunStatus](in.Status); err != nil {
		return nil, err
	}
	return out, nil
}

func EnvironmentToV1Beta1(in *alpha.Environment) (*beta.Environment, error) {
	out := &beta.Environment{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy(), Spec: beta.EnvironmentSpec{ProjectRef: in.Spec.ProjectRef, DisplayName: in.Spec.DisplayName, TargetNamespace: in.Spec.Namespace, Type: in.Spec.Type, RequiresApproval: in.Spec.RequiresApproval, Provider: in.Spec.GitOps, Policy: in.Spec.Policy}}
	out.TypeMeta.APIVersion = beta.GroupVersion.String()
	var err error
	out.Status, err = copyJSON[beta.EnvironmentStatus](in.Status)
	return out, err
}

func EnvironmentToV1Alpha1(in *beta.Environment) (*alpha.Environment, error) {
	out := &alpha.Environment{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy(), Spec: alpha.EnvironmentSpec{ProjectRef: in.Spec.ProjectRef, DisplayName: in.Spec.DisplayName, Namespace: in.Spec.TargetNamespace, Type: in.Spec.Type, RequiresApproval: in.Spec.RequiresApproval, GitOps: in.Spec.Provider, Policy: in.Spec.Policy}}
	out.TypeMeta.APIVersion = alpha.GroupVersion.String()
	var err error
	out.Status, err = copyJSON[alpha.EnvironmentStatus](in.Status)
	return out, err
}

func ReleaseToV1Beta1(in *alpha.Release) (*beta.Release, error) {
	out := &beta.Release{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy(), Spec: beta.ReleaseSpec{ProjectRef: in.Spec.ProjectRef, EnvironmentRef: in.Spec.EnvironmentRef, BuildRunRef: in.Spec.BuildRunRef, Artifact: in.Spec.Image, ApprovalPolicy: beta.ApprovalPolicy{Required: in.Spec.Approval.Required}, Strategy: in.Spec.Strategy, PromotionMode: in.Spec.PromotionMode, PullRequest: in.Spec.PullRequest, DeploymentTimeout: in.Spec.DeploymentTimeout, PromotedFrom: in.Spec.PromotedFrom, RollbackOf: in.Spec.RollbackOf, RollbackTo: in.Spec.RollbackTo}}
	out.TypeMeta.APIVersion = beta.GroupVersion.String()
	approval := in.Spec.Approval
	if approval.ApprovedBy != "" || approval.ApprovedAt != nil || approval.RejectedBy != "" || approval.RejectedAt != nil || approval.Comment != "" {
		out.Spec.LegacyApprovalDecision = &beta.LegacyApprovalDecision{ApprovedBy: approval.ApprovedBy, ApprovedAt: approval.ApprovedAt, RejectedBy: approval.RejectedBy, RejectedAt: approval.RejectedAt, Comment: approval.Comment}
	}
	var err error
	out.Status, err = copyJSON[beta.ReleaseStatus](in.Status)
	return out, err
}

func ReleaseToV1Alpha1(in *beta.Release) (*alpha.Release, error) {
	out := &alpha.Release{TypeMeta: in.TypeMeta, ObjectMeta: *in.ObjectMeta.DeepCopy(), Spec: alpha.ReleaseSpec{ProjectRef: in.Spec.ProjectRef, EnvironmentRef: in.Spec.EnvironmentRef, BuildRunRef: in.Spec.BuildRunRef, Image: in.Spec.Artifact, Approval: alpha.ReleaseApprovalSpec{Required: in.Spec.ApprovalPolicy.Required}, Strategy: in.Spec.Strategy, PromotionMode: in.Spec.PromotionMode, PullRequest: in.Spec.PullRequest, DeploymentTimeout: in.Spec.DeploymentTimeout, PromotedFrom: in.Spec.PromotedFrom, RollbackOf: in.Spec.RollbackOf, RollbackTo: in.Spec.RollbackTo}}
	out.TypeMeta.APIVersion = alpha.GroupVersion.String()
	if value := in.Spec.LegacyApprovalDecision; value != nil {
		out.Spec.Approval.ApprovedBy, out.Spec.Approval.ApprovedAt, out.Spec.Approval.RejectedBy, out.Spec.Approval.RejectedAt, out.Spec.Approval.Comment = value.ApprovedBy, value.ApprovedAt, value.RejectedBy, value.RejectedAt, value.Comment
	}
	var err error
	out.Status, err = copyJSON[alpha.ReleaseStatus](in.Status)
	return out, err
}

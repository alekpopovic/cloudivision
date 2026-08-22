package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CanTriggerBuild                = "CanTriggerBuild"
	CanUseSecret                   = "CanUseSecret"
	CanDeployToEnvironment         = "CanDeployToEnvironment"
	CanPromoteRelease              = "CanPromoteRelease"
	ImageMustHaveDigest            = "ImageMustHaveDigest"
	LatestImageForbidden           = "LatestImageForbidden"
	ImageMustBeSigned              = "ImageMustBeSigned"
	SBOMRequired                   = "SBOMRequired"
	CriticalVulnerabilitiesBlocked = "CriticalVulnerabilitiesBlocked"
	ProductionRequiresApproval     = "ProductionRequiresApproval"
	RunnerPrivilegedForbidden      = "RunnerPrivilegedForbidden"
	DockerSocketForbidden          = "DockerSocketForbidden"
	HostPathForbidden              = "HostPathForbidden"
	UnknownProviderForbidden       = "UnknownProviderForbidden"
)

type Decision struct {
	Allowed     bool        `json:"allowed"`
	Reason      string      `json:"reason"`
	Message     string      `json:"message"`
	EvaluatedAt time.Time   `json:"evaluatedAt"`
	Violations  []Violation `json:"violations,omitempty"`
}

type Violation struct {
	Policy    string `json:"policy"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	FieldPath string `json:"fieldPath,omitempty"`
}

type BuildRunPolicyInput struct {
	BuildRun             *cicdv1alpha1.BuildRun
	Project              *cicdv1alpha1.Project
	Repository           *cicdv1alpha1.Repository
	PipelineTemplate     *cicdv1alpha1.PipelineTemplate
	EnforceAuthorization bool
	TriggerAuthorized    bool
	SecretUseAuthorized  bool
}

type ReleasePolicyInput struct {
	Release     *cicdv1alpha1.Release
	BuildRun    *cicdv1alpha1.BuildRun
	Environment *cicdv1alpha1.Environment
}

type PipelineTemplatePolicyInput struct {
	PipelineTemplate     *cicdv1alpha1.PipelineTemplate
	EnforceAuthorization bool
	SecretUseAuthorized  bool
}

type Evaluator interface {
	EvaluateBuildRun(context.Context, BuildRunPolicyInput) Decision
	EvaluateRelease(context.Context, ReleasePolicyInput) Decision
	EvaluatePipelineTemplate(context.Context, PipelineTemplatePolicyInput) Decision
}

type DefaultEvaluator struct {
	Now func() time.Time
}

func NewDefaultEvaluator() *DefaultEvaluator {
	return &DefaultEvaluator{Now: time.Now}
}

func (e *DefaultEvaluator) EvaluateBuildRun(ctx context.Context, input BuildRunPolicyInput) Decision {
	_ = ctx
	violations := make([]Violation, 0)
	if input.EnforceAuthorization && !input.TriggerAuthorized {
		violations = append(violations, violation(CanTriggerBuild, "caller is not authorized to trigger builds", "spec.triggeredBy"))
	}
	if input.BuildRun == nil {
		violations = append(violations, violation(CanTriggerBuild, "BuildRun is required for policy evaluation", "buildRun"))
	}
	if input.Repository != nil && !knownRepositoryProvider(input.Repository.Spec.Provider) {
		violations = append(violations, violation(UnknownProviderForbidden, fmt.Sprintf("repository provider %q is not registered", input.Repository.Spec.Provider), "repository.spec.provider"))
	}
	if input.Project != nil && input.BuildRun != nil && input.Project.Name != input.BuildRun.Spec.ProjectRef {
		violations = append(violations, violation(CanTriggerBuild, "BuildRun project does not match the loaded Project", "spec.projectRef"))
	}
	if input.Repository != nil && input.BuildRun != nil && input.Repository.Spec.ProjectRef != input.BuildRun.Spec.ProjectRef {
		violations = append(violations, violation(CanTriggerBuild, "Repository belongs to a different project", "spec.repositoryRef"))
	}
	if input.PipelineTemplate != nil {
		templateDecision := e.EvaluatePipelineTemplate(ctx, PipelineTemplatePolicyInput{
			PipelineTemplate:     input.PipelineTemplate,
			EnforceAuthorization: input.EnforceAuthorization,
			SecretUseAuthorized:  input.SecretUseAuthorized,
		})
		violations = append(violations, templateDecision.Violations...)
	}
	return e.decision(violations)
}

func (e *DefaultEvaluator) EvaluatePipelineTemplate(_ context.Context, input PipelineTemplatePolicyInput) Decision {
	violations := make([]Violation, 0)
	template := input.PipelineTemplate
	if template == nil {
		return e.decision(violations)
	}
	if template.Spec.Security.AllowPrivileged {
		violations = append(violations, violation(RunnerPrivilegedForbidden, "privileged runners are forbidden by the default policy", "spec.security.allowPrivileged"))
	}
	if template.Spec.Build.Enabled && !knownBuilder(template.Spec.Build.Builder) {
		violations = append(violations, violation(UnknownProviderForbidden, fmt.Sprintf("build provider %q is not registered", template.Spec.Build.Builder), "spec.build.builder"))
	}
	if templateUsesSecret(template) && input.EnforceAuthorization && !input.SecretUseAuthorized {
		violations = append(violations, violation(CanUseSecret, "caller is not authorized to use secrets", "spec"))
	}
	for index, step := range template.Spec.Steps {
		values := append(append(append([]string{}, step.Command...), step.Args...), step.WorkingDir, step.Image)
		for _, value := range values {
			lower := strings.ToLower(value)
			if strings.Contains(lower, "/var/run/docker.sock") || strings.Contains(lower, "docker.sock") {
				violations = append(violations, violation(DockerSocketForbidden, "mounting or using the Docker socket is forbidden", fmt.Sprintf("spec.steps[%d]", index)))
				break
			}
			if strings.Contains(lower, "hostpath") || strings.Contains(lower, "host-path") {
				violations = append(violations, violation(HostPathForbidden, "hostPath volumes are forbidden", fmt.Sprintf("spec.steps[%d]", index)))
				break
			}
		}
	}
	return e.decision(violations)
}

func (e *DefaultEvaluator) EvaluateRelease(_ context.Context, input ReleasePolicyInput) Decision {
	violations := make([]Violation, 0)
	if input.Release == nil || input.Environment == nil || input.BuildRun == nil {
		return e.decision(append(violations, violation(CanPromoteRelease, "Release, BuildRun and Environment are required", "release")))
	}
	release, buildRun, environment := input.Release, input.BuildRun, input.Environment
	if release.Spec.ProjectRef != environment.Spec.ProjectRef || release.Spec.ProjectRef != buildRun.Spec.ProjectRef {
		violations = append(violations, violation(CanDeployToEnvironment, "Release, BuildRun and Environment must belong to the same project", "spec.projectRef"))
	}
	if buildRun.Status.Phase != "" && buildRun.Status.Phase != cicdv1alpha1.BuildRunPhaseSucceeded {
		violations = append(violations, violation(CanPromoteRelease, "only a successful BuildRun can be promoted", "spec.buildRunRef"))
	}
	if !knownGitOpsProvider(environment.Spec.GitOps.Provider) {
		violations = append(violations, violation(UnknownProviderForbidden, fmt.Sprintf("GitOps provider %q is not registered", environment.Spec.GitOps.Provider), "environment.spec.gitOps.provider"))
	}
	if environment.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction && release.Spec.Approval.ApprovedBy == "" {
		violations = append(violations, violation(ProductionRequiresApproval, "production releases require approval", "spec.approval.approvedBy"))
	}
	policy := environment.Spec.Policy
	if policy.RequireImageDigest && release.Spec.Image.Digest == "" {
		violations = append(violations, violation(ImageMustHaveDigest, "this environment requires an immutable image digest", "spec.image.digest"))
	}
	if environment.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction && !policy.AllowLatest && strings.EqualFold(release.Spec.Image.Tag, "latest") {
		violations = append(violations, violation(LatestImageForbidden, "production releases forbid the latest image tag unless explicitly allowed", "spec.image.tag"))
	}
	if policy.RequireSignedImages && buildRun.Status.SupplyChain.SignatureRef == "" {
		violations = append(violations, violation(ImageMustBeSigned, "this environment requires a signed image", "buildRun.status.supplyChain.signatureRef"))
	}
	if policy.RequireSBOM && buildRun.Status.SupplyChain.SBOMPath == "" && buildRun.Status.SupplyChain.SBOMDigest == "" {
		violations = append(violations, violation(SBOMRequired, "this environment requires SBOM metadata", "buildRun.status.supplyChain"))
	}
	if policy.BlockCriticalVulnerabilities && (buildRun.Status.SupplyChain.ScannerResultsRef == "" || buildRun.Status.SupplyChain.CriticalVulnerabilities > 0) {
		violations = append(violations, violation(CriticalVulnerabilitiesBlocked, fmt.Sprintf("critical vulnerabilities are blocked; scanner reported %d", buildRun.Status.SupplyChain.CriticalVulnerabilities), "buildRun.status.supplyChain.criticalVulnerabilities"))
	}
	return e.decision(violations)
}

func (e *DefaultEvaluator) decision(violations []Violation) Decision {
	now := time.Now()
	if e != nil && e.Now != nil {
		now = e.Now()
	}
	decision := Decision{Allowed: len(violations) == 0, Reason: "PolicyAllowed", Message: "policy evaluation allowed the operation", EvaluatedAt: now, Violations: violations}
	if len(violations) > 0 {
		decision.Reason = "PolicyDenied"
		decision.Message = violations[0].Message
	}
	return decision
}

func violation(name, message, fieldPath string) Violation {
	return Violation{Policy: name, Severity: "error", Message: message, FieldPath: fieldPath}
}

func templateUsesSecret(template *cicdv1alpha1.PipelineTemplate) bool {
	if template.Spec.SupplyChain.SigningKeySecretRef != nil {
		return true
	}
	for _, step := range template.Spec.Steps {
		for _, env := range step.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
				return true
			}
		}
	}
	return false
}

func knownRepositoryProvider(provider cicdv1alpha1.RepositoryProvider) bool {
	switch provider {
	case cicdv1alpha1.RepositoryProviderGitHub, cicdv1alpha1.RepositoryProviderGitLab, cicdv1alpha1.RepositoryProviderGitea, cicdv1alpha1.RepositoryProviderGeneric:
		return true
	default:
		return false
	}
}

func knownGitOpsProvider(provider cicdv1alpha1.GitOpsProvider) bool {
	switch provider {
	case cicdv1alpha1.GitOpsProviderArgoCD, cicdv1alpha1.GitOpsProviderFlux, cicdv1alpha1.GitOpsProviderGeneric:
		return true
	default:
		return false
	}
}

func knownBuilder(builder cicdv1alpha1.BuildBuilder) bool {
	switch builder {
	case cicdv1alpha1.BuildBuilderBuildKit, cicdv1alpha1.BuildBuilderBuildah, cicdv1alpha1.BuildBuilderNone:
		return true
	default:
		return false
	}
}

func ToStatus(decision Decision) cicdv1alpha1.PolicyDecisionStatus {
	evaluatedAt := metav1.NewTime(decision.EvaluatedAt)
	status := cicdv1alpha1.PolicyDecisionStatus{Allowed: decision.Allowed, Reason: decision.Reason, Message: decision.Message, EvaluatedAt: &evaluatedAt}
	for _, item := range decision.Violations {
		status.Violations = append(status.Violations, cicdv1alpha1.PolicyViolationStatus{Policy: item.Policy, Severity: item.Severity, Message: item.Message, FieldPath: item.FieldPath})
	}
	return status
}

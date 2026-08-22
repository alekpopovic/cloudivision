package policy

import (
	"context"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

func TestProductionReleaseRequiresApproval(t *testing.T) {
	input := safeReleaseInput()
	input.Environment.Spec.Type = cicdv1alpha1.EnvironmentTypeProduction
	assertViolation(t, NewDefaultEvaluator().EvaluateRelease(context.Background(), input), ProductionRequiresApproval)
}

func TestProductionReleaseRequiresDigestWhenConfigured(t *testing.T) {
	input := safeReleaseInput()
	input.Environment.Spec.Policy.RequireImageDigest = true
	assertViolation(t, NewDefaultEvaluator().EvaluateRelease(context.Background(), input), ImageMustHaveDigest)
}

func TestProductionReleaseBlocksLatestByDefault(t *testing.T) {
	input := safeReleaseInput()
	input.Environment.Spec.Type = cicdv1alpha1.EnvironmentTypeProduction
	input.Release.Spec.Approval.ApprovedBy = "release-manager"
	input.Release.Spec.Image.Tag = "latest"
	assertViolation(t, NewDefaultEvaluator().EvaluateRelease(context.Background(), input), LatestImageForbidden)
	input.Environment.Spec.Policy.AllowLatest = true
	decision := NewDefaultEvaluator().EvaluateRelease(context.Background(), input)
	if !decision.Allowed {
		t.Fatalf("allowLatest decision = %#v, want allowed", decision)
	}
}

func TestUnsignedImageDeniedWhenConfigured(t *testing.T) {
	input := safeReleaseInput()
	input.Environment.Spec.Policy.RequireSignedImages = true
	assertViolation(t, NewDefaultEvaluator().EvaluateRelease(context.Background(), input), ImageMustBeSigned)
}

func TestCriticalVulnerabilitiesDeniedWhenConfigured(t *testing.T) {
	input := safeReleaseInput()
	input.Environment.Spec.Policy.BlockCriticalVulnerabilities = true
	input.BuildRun.Status.SupplyChain.ScannerResultsRef = "scanner://result"
	input.BuildRun.Status.SupplyChain.CriticalVulnerabilities = 2
	assertViolation(t, NewDefaultEvaluator().EvaluateRelease(context.Background(), input), CriticalVulnerabilitiesBlocked)
}

func TestPrivilegedRunnerDeniedByDefault(t *testing.T) {
	template := &cicdv1alpha1.PipelineTemplate{Spec: cicdv1alpha1.PipelineTemplateSpec{Security: cicdv1alpha1.PipelineSecuritySpec{AllowPrivileged: true}}}
	assertViolation(t, NewDefaultEvaluator().EvaluatePipelineTemplate(context.Background(), PipelineTemplatePolicyInput{PipelineTemplate: template}), RunnerPrivilegedForbidden)
}

func TestSafeBuildRunAllowed(t *testing.T) {
	input := BuildRunPolicyInput{
		BuildRun:             &cicdv1alpha1.BuildRun{Spec: cicdv1alpha1.BuildRunSpec{ProjectRef: "project"}},
		Project:              &cicdv1alpha1.Project{},
		Repository:           &cicdv1alpha1.Repository{Spec: cicdv1alpha1.RepositorySpec{ProjectRef: "project", Provider: cicdv1alpha1.RepositoryProviderGitHub}},
		PipelineTemplate:     &cicdv1alpha1.PipelineTemplate{Spec: cicdv1alpha1.PipelineTemplateSpec{Build: cicdv1alpha1.PipelineBuildSpec{Builder: cicdv1alpha1.BuildBuilderNone}}},
		EnforceAuthorization: true,
		TriggerAuthorized:    true,
		SecretUseAuthorized:  true,
	}
	input.Project.Name = "project"
	decision := NewDefaultEvaluator().EvaluateBuildRun(context.Background(), input)
	if !decision.Allowed || len(decision.Violations) != 0 {
		t.Fatalf("decision = %#v, want allowed", decision)
	}
}

func safeReleaseInput() ReleasePolicyInput {
	return ReleasePolicyInput{
		Release: &cicdv1alpha1.Release{Spec: cicdv1alpha1.ReleaseSpec{
			ProjectRef: "project",
			Image:      cicdv1alpha1.ImageRef{Repository: "ghcr.io/acme/app", Tag: "main"},
		}},
		BuildRun: &cicdv1alpha1.BuildRun{
			Spec:   cicdv1alpha1.BuildRunSpec{ProjectRef: "project"},
			Status: cicdv1alpha1.BuildRunStatus{Phase: cicdv1alpha1.BuildRunPhaseSucceeded},
		},
		Environment: &cicdv1alpha1.Environment{Spec: cicdv1alpha1.EnvironmentSpec{
			ProjectRef: "project",
			Type:       cicdv1alpha1.EnvironmentTypeDev,
			GitOps:     cicdv1alpha1.EnvironmentGitOpsSpec{Provider: cicdv1alpha1.GitOpsProviderArgoCD},
		}},
	}
}

func assertViolation(t *testing.T, decision Decision, policyName string) {
	t.Helper()
	if decision.Allowed {
		t.Fatalf("decision allowed, want %s denial", policyName)
	}
	for _, item := range decision.Violations {
		if item.Policy == policyName {
			return
		}
	}
	t.Fatalf("violations = %#v, want %s", decision.Violations, policyName)
}

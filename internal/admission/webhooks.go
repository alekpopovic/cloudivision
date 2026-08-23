package admission

import (
	"context"
	"fmt"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	cradmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const AllowPrivilegedAnnotation = "cicd.cloudivision.io/allow-privileged"

type Defaulter struct{}

func (Defaulter) Default(_ context.Context, obj runtime.Object) error {
	switch value := obj.(type) {
	case *cicdv1alpha1.Project:
		if value.Spec.DefaultBranch == "" {
			value.Spec.DefaultBranch = "main"
		}
	case *cicdv1alpha1.Repository:
		if value.Spec.DefaultBranch == "" {
			value.Spec.DefaultBranch = "main"
		}
	case *cicdv1alpha1.PipelineTemplate:
		if value.Spec.Build.Enabled {
			if value.Spec.Build.ContextDir == "" {
				value.Spec.Build.ContextDir = "."
			}
			if value.Spec.Build.Dockerfile == "" {
				value.Spec.Build.Dockerfile = "Dockerfile"
			}
			if value.Spec.Build.Builder == "" {
				value.Spec.Build.Builder = cicdv1alpha1.BuildBuilderBuildKit
			}
		}
		if !value.Spec.Security.AllowPrivileged {
			value.Spec.Security.RunAsNonRoot = true
		}
	case *cicdv1alpha1.BuildRun:
		if value.Spec.Executor == "" {
			value.Spec.Executor = cicdv1alpha1.ExecutorTypeJob
		}
	case *cicdv1alpha1.Environment:
		if value.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction {
			value.Spec.RequiresApproval = true
		}
	case *cicdv1alpha1.Release:
		if value.Spec.Strategy == "" {
			value.Spec.Strategy = cicdv1alpha1.ReleaseStrategyGitOps
		}
		if value.Spec.PromotionMode == "" {
			value.Spec.PromotionMode = cicdv1alpha1.PromotionModeDirectCommit
		}
	default:
		return fmt.Errorf("unsupported admission type %T", obj)
	}
	return nil
}

type Validator struct{ Reader client.Reader }

func (v Validator) ValidateCreate(ctx context.Context, obj runtime.Object) (cradmission.Warnings, error) {
	return nil, v.validate(ctx, obj)
}
func (v Validator) ValidateUpdate(ctx context.Context, _, obj runtime.Object) (cradmission.Warnings, error) {
	return nil, v.validate(ctx, obj)
}
func (v Validator) ValidateDelete(context.Context, runtime.Object) (cradmission.Warnings, error) {
	return nil, nil
}

func (v Validator) validate(ctx context.Context, obj runtime.Object) error {
	switch value := obj.(type) {
	case *cicdv1alpha1.Project:
		if strings.TrimSpace(value.Spec.Namespace) == "" {
			return fmt.Errorf("spec.namespace is required")
		}
	case *cicdv1alpha1.Repository:
		if value.Spec.Webhook.Enabled && (value.Spec.Webhook.SecretRef.Name == "" || value.Spec.Webhook.SecretRef.Key == "") {
			return fmt.Errorf("spec.webhook.secretRef.name and key are required when webhook is enabled")
		}
	case *cicdv1alpha1.PipelineTemplate:
		if value.Spec.Security.AllowPrivileged && value.Annotations[AllowPrivilegedAnnotation] != "true" {
			return fmt.Errorf("spec.security.allowPrivileged requires annotation %s=true", AllowPrivilegedAnnotation)
		}
		if len(value.Spec.Steps) == 0 && !value.Spec.Build.Enabled {
			return fmt.Errorf("spec must contain at least one step or an enabled image build")
		}
	case *cicdv1alpha1.BuildRun:
		if value.Spec.GitOps.Enabled {
			if value.Spec.GitOps.Strategy == "" {
				return fmt.Errorf("spec.gitOps.strategy is required when GitOps is enabled")
			}
			if value.Spec.GitOps.EnvironmentRef == "" {
				return fmt.Errorf("spec.gitOps.environmentRef is required when GitOps is enabled")
			}
			if value.Spec.GitOps.RepoURL == "" {
				return fmt.Errorf("spec.gitOps.repoURL is required when GitOps is enabled")
			}
		}
	case *cicdv1alpha1.Environment:
		if value.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction && !value.Spec.RequiresApproval {
			return fmt.Errorf("production environments must require approval")
		}
	case *cicdv1alpha1.Release:
		if value.Spec.EnvironmentRef == "" {
			return fmt.Errorf("spec.environmentRef is required")
		}
		if v.Reader != nil {
			environment := &cicdv1alpha1.Environment{}
			if err := v.Reader.Get(ctx, client.ObjectKey{Namespace: value.Namespace, Name: value.Spec.EnvironmentRef}, environment); err != nil {
				return fmt.Errorf("read release environment %q: %w", value.Spec.EnvironmentRef, err)
			}
			if environment.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction && environment.Spec.Policy.RequireImageDigest && value.Spec.Image.Digest == "" {
				return fmt.Errorf("spec.image.digest is required by production environment policy")
			}
		}
	default:
		return fmt.Errorf("unsupported admission type %T", obj)
	}
	return nil
}

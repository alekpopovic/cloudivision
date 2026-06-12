package runner

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/build"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/executor/steps"
	cloudivisiongit "github.com/cloudivision/cloudivision/internal/git"
	"github.com/cloudivision/cloudivision/internal/observability"
	"github.com/cloudivision/cloudivision/internal/redact"
	"github.com/cloudivision/cloudivision/internal/supplychain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionRepositoryCloned = "RepositoryCloned"
	ConditionStepsCompleted   = "StepsCompleted"
	ConditionImageBuilt       = "ImageBuilt"
	ConditionImagePushed      = "ImagePushed"
	ConditionSupplyChainReady = "SupplyChainReady"
)

type StepRunner interface {
	Run(ctx context.Context, sourceDir string, pipelineSteps []cicdv1alpha1.PipelineStep, redactor redact.Redactor) error
}

type Runner struct {
	Client     client.Client
	Git        cloudivisiongit.Client
	Steps      StepRunner
	Builder    build.ImageBuilder
	SBOM       supplychain.SBOMGenerator
	Scanner    supplychain.VulnerabilityScanner
	Signer     supplychain.ImageSigner
	Provenance supplychain.ProvenanceWriter
	Workspace  string
	Logger     *slog.Logger
}

func New(k8sClient client.Client, logger *slog.Logger) Runner {
	return Runner{
		Client:     k8sClient,
		Git:        cloudivisiongit.ExecClient{},
		Steps:      steps.Runner{Output: os.Stdout, Logger: logger},
		Builder:    build.BuildKitBuilder{},
		SBOM:       supplychain.NoopSBOMGenerator{},
		Scanner:    supplychain.NoopScanner{},
		Signer:     supplychain.NoopSigner{},
		Provenance: supplychain.NoopProvenanceWriter{},
		Workspace:  "/workspace",
		Logger:     logger,
	}
}

func (r Runner) Run(ctx context.Context, cfg Config) error {
	if r.Git == nil {
		r.Git = cloudivisiongit.ExecClient{}
	}
	if r.Steps == nil {
		r.Steps = steps.Runner{Output: os.Stdout, Logger: r.Logger}
	}
	if r.Builder == nil {
		r.Builder = build.BuildKitBuilder{}
	}
	if r.SBOM == nil {
		r.SBOM = supplychain.NoopSBOMGenerator{}
	}
	if r.Scanner == nil {
		r.Scanner = supplychain.NoopScanner{}
	}
	if r.Signer == nil {
		r.Signer = supplychain.NoopSigner{}
	}
	if r.Provenance == nil {
		r.Provenance = supplychain.NoopProvenanceWriter{}
	}
	if r.Workspace == "" {
		r.Workspace = "/workspace"
	}

	buildRun := &cicdv1alpha1.BuildRun{}
	key := types.NamespacedName{Name: cfg.BuildRunName, Namespace: cfg.BuildRunNamespace}
	if err := r.Client.Get(ctx, key, buildRun); err != nil {
		return fmt.Errorf("load BuildRun %s: %w", key, err)
	}
	logger := r.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With(
		"namespace", buildRun.Namespace,
		"buildRun", buildRun.Name,
		"project", buildRun.Spec.ProjectRef,
		"correlationId", buildRun.Annotations[observability.CorrelationIDAnno],
	)
	if stepRunner, ok := r.Steps.(steps.Runner); ok && stepRunner.Logger == nil {
		stepRunner.Logger = logger
		r.Steps = stepRunner
	}
	repository := &cicdv1alpha1.Repository{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: buildRun.Spec.RepositoryRef, Namespace: cfg.BuildRunNamespace}, repository); err != nil {
		return r.fail(ctx, buildRun, "RepositoryLoadFailed", fmt.Sprintf("load Repository %q: %v", buildRun.Spec.RepositoryRef, err))
	}
	template := &cicdv1alpha1.PipelineTemplate{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: buildRun.Spec.PipelineTemplateRef, Namespace: cfg.BuildRunNamespace}, template); err != nil {
		return r.fail(ctx, buildRun, "PipelineTemplateLoadFailed", fmt.Sprintf("load PipelineTemplate %q: %v", buildRun.Spec.PipelineTemplateRef, err))
	}
	redactor := redact.FromEnv(secretValues(buildRun, repository, template))

	now := metav1.Now()
	if err := domain.MarkBuildRunStarted(buildRun, now); err != nil {
		return fmt.Errorf("mark BuildRun started: %w", err)
	}
	if err := r.updateStatus(ctx, buildRun); err != nil {
		return err
	}

	sourceDir := filepath.Join(r.Workspace, "source")
	if err := os.RemoveAll(sourceDir); err != nil {
		return r.fail(ctx, buildRun, "WorkspaceCleanupFailed", err.Error())
	}
	if err := os.MkdirAll(r.Workspace, 0o755); err != nil {
		return r.fail(ctx, buildRun, "WorkspaceCreateFailed", err.Error())
	}
	if err := r.Git.Clone(ctx, repository.Spec.URL, sourceDir); err != nil {
		return r.fail(ctx, buildRun, "RepositoryCloneFailed", redactor.Mask(err.Error()))
	}
	if err := r.Git.Checkout(ctx, sourceDir, checkoutRef(buildRun, cfg)); err != nil {
		return r.fail(ctx, buildRun, "RepositoryCheckoutFailed", redactor.Mask(err.Error()))
	}
	r.setCondition(buildRun, ConditionRepositoryCloned, metav1.ConditionTrue, "RepositoryCloned", "Repository cloned and checked out.")
	if err := r.updateStatus(ctx, buildRun); err != nil {
		return err
	}

	if err := r.Steps.Run(ctx, sourceDir, template.Spec.Steps, redactor); err != nil {
		return r.fail(ctx, buildRun, "StepFailed", redactor.Mask(err.Error()))
	}
	r.setCondition(buildRun, ConditionStepsCompleted, metav1.ConditionTrue, "StepsCompleted", "Pipeline steps completed.")
	if err := r.updateStatus(ctx, buildRun); err != nil {
		return err
	}

	result := &build.BuildResult{
		ImageRepository: buildRun.Spec.Image.Repository,
		Tag:             buildRun.Spec.Image.Tag,
		Digest:          buildRun.Spec.Image.Digest,
	}
	if template.Spec.Build.Enabled {
		req := build.BuildRequest{
			ContextDir:      filepath.Join(sourceDir, template.Spec.Build.ContextDir),
			Dockerfile:      template.Spec.Build.Dockerfile,
			ImageRepository: buildRun.Spec.Image.Repository,
			ImageTag:        buildRun.Spec.Image.Tag,
			Push:            template.Spec.Build.Push,
			Env:             buildParamsEnv(buildRun.Spec.Params),
		}
		buildResult, err := r.Builder.Build(ctx, req)
		if err != nil {
			return r.fail(ctx, buildRun, "ImageBuildFailed", redactor.Mask(err.Error()))
		}
		result = buildResult
		r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionTrue, "ImageBuilt", "Image build completed.")
		if template.Spec.Build.Push {
			r.setCondition(buildRun, ConditionImagePushed, metav1.ConditionTrue, "ImagePushed", "Image push completed.")
		}
		if err := r.updateStatus(ctx, buildRun); err != nil {
			return err
		}
		if err := r.runSupplyChainHooks(ctx, buildRun, template, sourceDir, result); err != nil {
			return r.fail(ctx, buildRun, "SupplyChainFailed", redactor.Mask(err.Error()))
		}
		if err := r.updateStatus(ctx, buildRun); err != nil {
			return err
		}
	}

	completed := metav1.Now()
	if err := domain.MarkBuildRunSucceeded(buildRun, completed, cicdv1alpha1.ImageRef{
		Repository: result.ImageRepository,
		Tag:        result.Tag,
		Digest:     result.Digest,
	}); err != nil {
		return fmt.Errorf("mark BuildRun succeeded: %w", err)
	}
	return r.updateStatus(ctx, buildRun)
}

func (r Runner) runSupplyChainHooks(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, template *cicdv1alpha1.PipelineTemplate, sourceDir string, result *build.BuildResult) error {
	if result == nil {
		result = &build.BuildResult{}
	}
	image := cicdv1alpha1.ImageRef{
		Repository: result.ImageRepository,
		Tag:        result.Tag,
		Digest:     result.Digest,
	}
	base := supplychain.ImageContext{
		BuildRunName: buildRun.Name,
		Namespace:    buildRun.Namespace,
		ProjectName:  buildRun.Spec.ProjectRef,
		SourceDir:    sourceDir,
		Image:        image,
	}
	status := buildRun.Status.SupplyChain
	if result.SBOMPath != "" {
		status.SBOMPath = result.SBOMPath
	}
	if template.Spec.SupplyChain.GenerateSBOM {
		sbom, err := r.SBOM.GenerateSBOM(ctx, supplychain.SBOMRequest{ImageContext: base})
		if err != nil {
			return fmt.Errorf("generate SBOM: %w", err)
		}
		if sbom != nil {
			if sbom.Path != "" {
				status.SBOMPath = sbom.Path
			}
			if sbom.Digest != "" {
				status.SBOMDigest = sbom.Digest
			}
		}
	}
	if template.Spec.SupplyChain.ScanImage {
		scan, err := r.Scanner.ScanImage(ctx, supplychain.ScanRequest{ImageContext: base, SBOMPath: status.SBOMPath})
		if err != nil {
			return fmt.Errorf("scan image: %w", err)
		}
		if scan != nil && scan.ResultsRef != "" {
			status.ScannerResultsRef = scan.ResultsRef
		}
	}
	if template.Spec.SupplyChain.SignImage {
		signature, err := r.Signer.SignImage(ctx, supplychain.SignRequest{ImageContext: base})
		if err != nil {
			return fmt.Errorf("sign image: %w", err)
		}
		if signature != nil && signature.SignatureRef != "" {
			status.SignatureRef = signature.SignatureRef
		}
	}
	if template.Spec.SupplyChain.GenerateSBOM || template.Spec.SupplyChain.ScanImage || template.Spec.SupplyChain.SignImage || result.Digest != "" {
		provenance, err := r.Provenance.WriteProvenance(ctx, supplychain.ProvenanceRequest{
			ImageContext:      base,
			SBOMPath:          status.SBOMPath,
			SBOMDigest:        status.SBOMDigest,
			SignatureRef:      status.SignatureRef,
			ScannerResultsRef: status.ScannerResultsRef,
		})
		if err != nil {
			return fmt.Errorf("write provenance: %w", err)
		}
		if provenance != nil && provenance.Ref != "" {
			status.ProvenanceRef = provenance.Ref
		}
	}
	buildRun.Status.SupplyChain = status
	if status.SBOMPath != "" || status.SBOMDigest != "" || status.SignatureRef != "" || status.ScannerResultsRef != "" || status.ProvenanceRef != "" {
		r.setCondition(buildRun, ConditionSupplyChainReady, metav1.ConditionTrue, "SupplyChainMetadataRecorded", "Supply-chain metadata has been recorded.")
	}
	return nil
}

func (r Runner) fail(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, reason, message string) error {
	if r.Logger != nil {
		r.Logger.Warn(
			"runner failed",
			"namespace", buildRun.Namespace,
			"buildRun", buildRun.Name,
			"project", buildRun.Spec.ProjectRef,
			"reason", reason,
			"message", redact.MaskString(message),
			"correlationId", buildRun.Annotations[observability.CorrelationIDAnno],
		)
	}
	now := metav1.Now()
	if err := domain.MarkBuildRunFailed(buildRun, now, reason, message); err != nil {
		return fmt.Errorf("%s: %s", reason, message)
	}
	if err := r.updateStatus(ctx, buildRun); err != nil {
		return err
	}
	return fmt.Errorf("%s: %s", reason, message)
}

func (r Runner) updateStatus(ctx context.Context, buildRun *cicdv1alpha1.BuildRun) error {
	if err := r.Client.Status().Update(ctx, buildRun); err != nil {
		return fmt.Errorf("update BuildRun status: %w", err)
	}
	return nil
}

func (r Runner) setCondition(buildRun *cicdv1alpha1.BuildRun, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	domain.SetCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: buildRun.Generation,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}

func checkoutRef(buildRun *cicdv1alpha1.BuildRun, cfg Config) string {
	if buildRun.Spec.CommitSHA != "" {
		return buildRun.Spec.CommitSHA
	}
	if buildRun.Spec.Revision != "" {
		return buildRun.Spec.Revision
	}
	if cfg.Revision != "" {
		return cfg.Revision
	}
	if buildRun.Spec.Branch != "" {
		return buildRun.Spec.Branch
	}
	return cfg.Branch
}

func buildParamsEnv(params map[string]string) map[string]string {
	env := map[string]string{}
	for key, value := range params {
		env["PARAM_"+key] = value
	}
	return env
}

func secretEnv(pipelineSteps []cicdv1alpha1.PipelineStep) map[string]string {
	env := map[string]string{}
	for _, step := range pipelineSteps {
		for _, item := range step.Env {
			env[item.Name] = item.Value
		}
	}
	return env
}

func secretValues(buildRun *cicdv1alpha1.BuildRun, repository *cicdv1alpha1.Repository, template *cicdv1alpha1.PipelineTemplate) map[string]string {
	values := secretEnv(template.Spec.Steps)
	for key, value := range buildParamsEnv(buildRun.Spec.Params) {
		values[key] = value
	}
	for key, value := range repositoryURLSecrets(repository.Spec.URL) {
		values[key] = value
	}
	return values
}

func repositoryURLSecrets(rawURL string) map[string]string {
	values := map[string]string{}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User == nil {
		return values
	}
	if username := parsed.User.Username(); username != "" {
		values["REPOSITORY_URL_USERNAME"] = username
	}
	if password, ok := parsed.User.Password(); ok && password != "" {
		values["REPOSITORY_URL_TOKEN"] = password
	}
	if userInfo := parsed.User.String(); userInfo != "" {
		values["REPOSITORY_URL_SECRET"] = userInfo
	}
	return values
}

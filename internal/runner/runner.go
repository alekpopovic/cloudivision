package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/artifacts"
	"github.com/cloudivision/cloudivision/internal/build"
	"github.com/cloudivision/cloudivision/internal/domain"
	"github.com/cloudivision/cloudivision/internal/executor/steps"
	cloudivisiongit "github.com/cloudivision/cloudivision/internal/git"
	"github.com/cloudivision/cloudivision/internal/kube"
	"github.com/cloudivision/cloudivision/internal/logstore"
	"github.com/cloudivision/cloudivision/internal/observability"
	providerregistry "github.com/cloudivision/cloudivision/internal/provider/registry"
	"github.com/cloudivision/cloudivision/internal/redact"
	"github.com/cloudivision/cloudivision/internal/supplychain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionRepositoryCloned  = "RepositoryCloned"
	ConditionStepsCompleted    = "StepsCompleted"
	ConditionImageBuildStarted = "ImageBuildStarted"
	ConditionImageBuilt        = "ImageBuilt"
	ConditionImagePushed       = "ImagePushed"
	ConditionImageDigest       = "ImageDigestCaptured"
	ConditionSupplyChainReady  = "SupplyChainReady"
	ConditionSBOMGenerated     = "SBOMGenerated"
	ConditionImageScanned      = "ImageScanned"
	ConditionImageSigned       = "ImageSigned"
	ConditionProvenanceWritten = "ProvenanceWritten"
)

type StepRunner interface {
	Run(ctx context.Context, sourceDir string, pipelineSteps []cicdv1alpha1.PipelineStep, redactor redact.Redactor) error
}

type Runner struct {
	Client        client.Client
	Git           cloudivisiongit.Client
	Steps         StepRunner
	Builder       build.ImageBuilder
	SBOM          supplychain.SBOMGenerator
	Scanner       supplychain.VulnerabilityScanner
	Signer        supplychain.ImageSigner
	Provenance    supplychain.ProvenanceWriter
	Workspace     string
	Logger        *slog.Logger
	LogStore      logstore.LogStore
	ArtifactStore artifacts.ArtifactStore
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
	r.configureSupplyChainAdapters(template.Spec.SupplyChain)
	redactor := redact.FromEnv(secretValues(buildRun, repository, template))
	if cfg.MaxLogBytes > 0 {
		if stepRunner, ok := r.Steps.(steps.Runner); ok {
			output := stepRunner.Output
			if output == nil {
				output = os.Stdout
			}
			stepRunner.Output = &boundedWriter{Writer: output, Remaining: cfg.MaxLogBytes}
			r.Steps = stepRunner
		}
	}
	var storedLogs logstore.LogStore
	if r.LogStore != nil {
		storedLogs = logstore.RedactingStore{Store: r.LogStore, Mask: redactor.Mask}
		backend := cfg.LogBackend
		if backend == "" {
			backend = "configured"
		}
		buildRun.Status.Log.Backend = backend
		buildRun.Status.Log.Ref = buildRun.Namespace + "/" + buildRun.Name
		storedBytes := int64(0)
		if stepRunner, ok := r.Steps.(steps.Runner); ok {
			stepRunner.Observe = func(step, output string) error {
				storedBytes += int64(len(output))
				if cfg.MaxLogBytes > 0 && storedBytes > cfg.MaxLogBytes {
					return fmt.Errorf("project maxLogSize of %d bytes exceeded", cfg.MaxLogBytes)
				}
				lines := []logstore.LogLine{}
				for _, message := range strings.Split(strings.TrimSuffix(output, "\n"), "\n") {
					if message != "" {
						lines = append(lines, logstore.LogLine{Timestamp: time.Now().UTC(), Step: step, Message: message})
					}
				}
				return storedLogs.Append(ctx, logstore.AppendLogRequest{Namespace: buildRun.Namespace, BuildRun: buildRun.Name, Lines: lines})
			}
			r.Steps = stepRunner
		}
		if err := storedLogs.Append(ctx, logstore.AppendLogRequest{Namespace: buildRun.Namespace, BuildRun: buildRun.Name, Lines: []logstore.LogLine{{Timestamp: time.Now().UTC(), Message: "Build runner started."}}}); err != nil {
			return r.fail(ctx, buildRun, "LogStoreWriteFailed", redactor.Mask(err.Error()))
		}
	}

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
	if r.ArtifactStore != nil {
		artifactStatuses, err := r.collectArtifacts(ctx, buildRun, template, sourceDir, cfg.MaxArtifactsBytes)
		if err != nil {
			return r.fail(ctx, buildRun, "ArtifactCollectionFailed", redactor.Mask(err.Error()))
		}
		buildRun.Status.Artifacts = artifactStatuses
	}
	if err := r.updateStatus(ctx, buildRun); err != nil {
		return err
	}

	result := &build.BuildResult{
		ImageRepository: buildRun.Spec.Image.Repository,
		Tag:             buildRun.Spec.Image.Tag,
		Digest:          buildRun.Spec.Image.Digest,
	}
	if template.Spec.Build.Enabled {
		r.setCondition(buildRun, ConditionImageBuildStarted, metav1.ConditionTrue, "ImageBuildStarted", "Image build started.")
		if err := r.updateStatus(ctx, buildRun); err != nil {
			return err
		}
		contextDir, err := build.ResolveContextDir(sourceDir, template.Spec.Build.ContextDir)
		if err != nil {
			reason := build.FailureReason(err)
			r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, reason, "Image build context is invalid.")
			return r.fail(ctx, buildRun, reason, redactor.Mask(err.Error()))
		}
		imageRepository := buildRun.Spec.Image.Repository
		imageTag := buildRun.Spec.Image.Tag
		buildEnv := buildParamsEnv(buildRun.Spec.Params)
		registryRedactor := redact.New()
		if cfg.RegistryProvider != "" {
			registryProvider, providerErr := providerregistry.New(cfg.RegistryProvider)
			if providerErr != nil {
				r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, "RegistryProviderUnsupported", "Configured registry provider is unsupported.")
				return r.fail(ctx, buildRun, "RegistryProviderUnsupported", redactor.Mask(providerErr.Error()))
			}
			resolved, resolveErr := registryProvider.ResolveImage(ctx, providerregistry.ImageRequest{
				ImagePrefix: cfg.RegistryImagePrefix,
				Repository:  imageRepository,
				Tag:         imageTag,
			})
			if resolveErr != nil {
				r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, "RegistryImageInvalid", "Configured registry image is invalid.")
				return r.fail(ctx, buildRun, "RegistryImageInvalid", redactor.Mask(resolveErr.Error()))
			}
			imageRepository = resolved.Repository
			imageTag = resolved.Tag
			if template.Spec.Build.Push {
				credential, credentialErr := providerregistry.LoadCredentialDir(cfg.RegistryCredentialsDir)
				if credentialErr != nil {
					reason := registryCredentialFailureReason(credentialErr)
					r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, reason, "Registry credentials could not be loaded.")
					return r.fail(ctx, buildRun, reason, redactor.Mask(credentialErr.Error()))
				}
				registryRedactor = providerregistry.Redactor(credential)
				registryHost, hostErr := providerregistry.RegistryHost(imageRepository)
				if hostErr != nil {
					r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, "RegistryImageInvalid", "Configured registry image has no registry host.")
					return r.fail(ctx, buildRun, "RegistryImageInvalid", redactor.Mask(hostErr.Error()))
				}
				login, loginErr := registryProvider.Login(ctx, providerregistry.LoginRequest{Registry: registryHost, Credential: credential})
				if loginErr != nil {
					reason := registryCredentialFailureReason(loginErr)
					r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, reason, "Registry login configuration failed.")
					return r.fail(ctx, buildRun, reason, redactor.Mask(registryRedactor.Mask(loginErr.Error())))
				}
				dockerConfigDir := filepath.Join(r.Workspace, ".docker")
				if removeErr := os.RemoveAll(dockerConfigDir); removeErr != nil {
					return r.fail(ctx, buildRun, "RegistryCredentialsInvalid", redactor.Mask(removeErr.Error()))
				}
				if _, writeErr := providerregistry.WriteDockerConfig(dockerConfigDir, login); writeErr != nil {
					return r.fail(ctx, buildRun, "RegistryCredentialsInvalid", redactor.Mask(registryRedactor.Mask(writeErr.Error())))
				}
				defer os.RemoveAll(dockerConfigDir)
				buildEnv["DOCKER_CONFIG"] = dockerConfigDir
			}
		}
		req := build.BuildRequest{
			ContextDir:      contextDir,
			Dockerfile:      template.Spec.Build.Dockerfile,
			ImageRepository: imageRepository,
			ImageTag:        imageTag,
			Push:            template.Spec.Build.Push,
			BuildArgs:       template.Spec.Build.BuildArgs,
			Target:          template.Spec.Build.Target,
			Platforms:       template.Spec.Build.Platforms,
			Labels:          template.Spec.Build.Labels,
			Cache: build.CacheConfig{
				Enabled: template.Spec.Build.Cache.Enabled,
				Mode:    build.CacheMode(template.Spec.Build.Cache.Mode),
				Ref:     template.Spec.Build.Cache.Ref,
			},
			Env: buildEnv,
		}
		buildResult, err := r.Builder.Build(ctx, req)
		if err != nil {
			reason := build.FailureReason(err)
			r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionFalse, reason, "Image build did not complete.")
			if reason == build.ReasonDigestCaptureFailed {
				r.setCondition(buildRun, ConditionImageDigest, metav1.ConditionFalse, reason, "BuildKit did not return a valid image digest.")
			}
			return r.fail(ctx, buildRun, reason, redactor.Mask(registryRedactor.Mask(err.Error())))
		}
		result = buildResult
		buildRun.Status.Image = &cicdv1alpha1.ImageRef{
			Repository: result.ImageRepository,
			Tag:        result.Tag,
			Digest:     result.Digest,
		}
		r.setCondition(buildRun, ConditionImageBuilt, metav1.ConditionTrue, "ImageBuilt", "Image build completed.")
		if template.Spec.Build.Push {
			r.setCondition(buildRun, ConditionImagePushed, metav1.ConditionTrue, "ImagePushed", "Image push completed.")
		} else {
			r.setCondition(buildRun, ConditionImagePushed, metav1.ConditionFalse, "PushDisabled", "Image push was disabled for this pipeline.")
		}
		if result.Digest != "" {
			r.setCondition(buildRun, ConditionImageDigest, metav1.ConditionTrue, "ImageDigestCaptured", "Image digest was captured from BuildKit metadata.")
		} else {
			r.setCondition(buildRun, ConditionImageDigest, metav1.ConditionFalse, "ImageDigestUnavailable", "BuildKit did not return an image digest.")
		}
		if err := r.updateStatus(ctx, buildRun); err != nil {
			return err
		}
		if err := r.runSupplyChainHooks(ctx, buildRun, repository, template, sourceDir, result); err != nil {
			reason := "SupplyChainFailed"
			hookErr := &supplyChainHookError{}
			if errors.As(err, &hookErr) {
				reason = hookErr.Reason
			}
			return r.fail(ctx, buildRun, reason, redactor.Mask(err.Error()))
		}
		if err := r.updateStatus(ctx, buildRun); err != nil {
			return err
		}
	}

	if storedLogs != nil {
		if err := storedLogs.Append(ctx, logstore.AppendLogRequest{Namespace: buildRun.Namespace, BuildRun: buildRun.Name, Lines: []logstore.LogLine{{Timestamp: time.Now().UTC(), Message: "Build runner completed successfully."}}}); err != nil {
			return r.fail(ctx, buildRun, "LogStoreWriteFailed", redactor.Mask(err.Error()))
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

func registryCredentialFailureReason(err error) string {
	if errors.Is(err, providerregistry.ErrCredentialsMissing) {
		return "RegistryCredentialsMissing"
	}
	if errors.Is(err, providerregistry.ErrUnsupportedProvider) {
		return "RegistryProviderUnsupported"
	}
	return "RegistryCredentialsInvalid"
}

type supplyChainHookError struct {
	Reason string
	Err    error
}

func (e *supplyChainHookError) Error() string { return e.Err.Error() }
func (e *supplyChainHookError) Unwrap() error { return e.Err }

func (r *Runner) configureSupplyChainAdapters(spec cicdv1alpha1.PipelineSupplyChainSpec) {
	if spec.SBOMAdapter == "syft" {
		r.SBOM = supplychain.SyftSBOMGenerator{}
	}
	if spec.ScannerAdapter == "grype" {
		r.Scanner = supplychain.GrypeScanner{}
	}
	if spec.SignerAdapter == "cosign" {
		keyPath := ""
		if spec.SigningKeySecretRef != nil {
			keyPath = "/var/run/secrets/cloudivision-signing/key"
		}
		r.Signer = supplychain.CosignSigner{Keyless: spec.CosignKeyless, KeyPath: keyPath}
	}
	if spec.ProvenanceAdapter == "json" {
		r.Provenance = supplychain.JSONProvenanceWriter{}
	}
}

func (r Runner) runSupplyChainHooks(ctx context.Context, buildRun *cicdv1alpha1.BuildRun, repository *cicdv1alpha1.Repository, template *cicdv1alpha1.PipelineTemplate, sourceDir string, result *build.BuildResult) error {
	if result == nil {
		result = &build.BuildResult{}
	}
	image := cicdv1alpha1.ImageRef{
		Repository: result.ImageRepository,
		Tag:        result.Tag,
		Digest:     result.Digest,
	}
	base := supplychain.ImageContext{
		BuildRunName:     buildRun.Name,
		Namespace:        buildRun.Namespace,
		ProjectName:      buildRun.Spec.ProjectRef,
		SourceDir:        sourceDir,
		RepositoryURL:    repository.Spec.URL,
		CommitSHA:        buildRun.Spec.CommitSHA,
		PipelineTemplate: buildRun.Spec.PipelineTemplateRef,
		Image:            image,
	}
	if base.CommitSHA == "" {
		base.CommitSHA = buildRun.Spec.Revision
	}
	if buildRun.Status.StartedAt != nil {
		startedAt := buildRun.Status.StartedAt.Time
		base.StartedAt = &startedAt
	}
	completedAt := time.Now().UTC()
	base.CompletedAt = &completedAt
	status := buildRun.Status.SupplyChain
	if result.SBOMPath != "" {
		status.SBOMPath = result.SBOMPath
	}
	if template.Spec.SupplyChain.GenerateSBOM {
		sbom, err := r.SBOM.GenerateSBOM(ctx, supplychain.SBOMRequest{ImageContext: base})
		if err != nil {
			r.setCondition(buildRun, ConditionSBOMGenerated, metav1.ConditionFalse, "SBOMGenerationFailed", err.Error())
			return &supplyChainHookError{Reason: "SBOMGenerationFailed", Err: fmt.Errorf("generate SBOM: %w", err)}
		}
		if sbom != nil {
			if sbom.Path != "" {
				status.SBOMPath = sbom.Path
			}
			if sbom.Digest != "" {
				status.SBOMDigest = sbom.Digest
			}
		}
		if status.SBOMPath != "" || status.SBOMDigest != "" {
			r.setCondition(buildRun, ConditionSBOMGenerated, metav1.ConditionTrue, "SBOMGenerated", "SBOM was generated and its digest recorded.")
		} else {
			r.setCondition(buildRun, ConditionSBOMGenerated, metav1.ConditionFalse, "NoopAdapter", "SBOM adapter returned no artifact.")
		}
		buildRun.Status.SupplyChain = status
	}
	if template.Spec.SupplyChain.ScanImage {
		scan, err := r.Scanner.ScanImage(ctx, supplychain.ScanRequest{ImageContext: base, SBOMPath: status.SBOMPath})
		if err != nil {
			r.setCondition(buildRun, ConditionImageScanned, metav1.ConditionFalse, "ImageScanFailed", err.Error())
			return &supplyChainHookError{Reason: "ImageScanFailed", Err: fmt.Errorf("scan image: %w", err)}
		}
		if scan != nil {
			status.ScannerResultsRef = scan.ResultsRef
			status.CriticalVulnerabilities = scan.Critical
			status.HighVulnerabilities = scan.High
			status.MediumVulnerabilities = scan.Medium
			status.LowVulnerabilities = scan.Low
		}
		if status.ScannerResultsRef != "" {
			r.setCondition(buildRun, ConditionImageScanned, metav1.ConditionTrue, "ImageScanned", "Vulnerability scan results and severity summary were recorded.")
		} else {
			r.setCondition(buildRun, ConditionImageScanned, metav1.ConditionFalse, "NoopAdapter", "Scanner adapter returned no results.")
		}
		buildRun.Status.SupplyChain = status
	}
	if template.Spec.SupplyChain.SignImage {
		signature, err := r.Signer.SignImage(ctx, supplychain.SignRequest{ImageContext: base})
		if err != nil {
			r.setCondition(buildRun, ConditionImageSigned, metav1.ConditionFalse, "ImageSigningFailed", err.Error())
			return &supplyChainHookError{Reason: "ImageSigningFailed", Err: fmt.Errorf("sign image: %w", err)}
		}
		if signature != nil && signature.SignatureRef != "" {
			status.SignatureRef = signature.SignatureRef
		}
		if status.SignatureRef != "" {
			r.setCondition(buildRun, ConditionImageSigned, metav1.ConditionTrue, "ImageSigned", "Image signature reference was recorded.")
		} else {
			r.setCondition(buildRun, ConditionImageSigned, metav1.ConditionFalse, "NoopAdapter", "Signer adapter returned no signature.")
		}
		buildRun.Status.SupplyChain = status
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
			r.setCondition(buildRun, ConditionProvenanceWritten, metav1.ConditionFalse, "ProvenanceWriteFailed", err.Error())
			return &supplyChainHookError{Reason: "ProvenanceWriteFailed", Err: fmt.Errorf("write provenance: %w", err)}
		}
		if provenance != nil && provenance.Ref != "" {
			status.ProvenanceRef = provenance.Ref
		}
		if status.ProvenanceRef != "" {
			r.setCondition(buildRun, ConditionProvenanceWritten, metav1.ConditionTrue, "ProvenanceWritten", "Build provenance reference was recorded.")
		} else {
			r.setCondition(buildRun, ConditionProvenanceWritten, metav1.ConditionFalse, "NoopAdapter", "Provenance adapter returned no artifact.")
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
	if err := kube.UpdateStatusWithRetry(ctx, r.Client, buildRun); err != nil {
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

package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/build"
	"github.com/cloudivision/cloudivision/internal/executor/steps"
	cloudivisiongit "github.com/cloudivision/cloudivision/internal/git"
	"github.com/cloudivision/cloudivision/internal/supplychain"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRunnerExecutesPipelineWithoutImageBuild(t *testing.T) {
	ctx := context.Background()
	repo := createGitRepository(t)
	buildRun := testBuildRun(repo)
	template := testPipelineTemplate([]cicdv1alpha1.PipelineStep{
		{
			Name:    "check",
			Image:   "alpine:3.20",
			Command: []string{"sh"},
			Args:    []string{"-c", "test -f README.md"},
		},
	})
	k8sClient := newFakeRunnerClient(t, buildRun, testRepository(repo), template)
	cfg := testConfig(repo)

	runner := Runner{
		Client:    k8sClient,
		Git:       cloudivisiongit.ExecClient{},
		Steps:     steps.Runner{},
		Builder:   failBuilder{},
		Workspace: filepath.Join(t.TempDir(), "workspace"),
	}
	if err := runner.Run(ctx, cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseSucceeded {
		t.Fatalf("phase = %q, want Succeeded", updated.Status.Phase)
	}
	assertCondition(t, updated.Status.Conditions, ConditionRepositoryCloned)
	assertCondition(t, updated.Status.Conditions, ConditionStepsCompleted)
	if updated.Status.Failure.Reason != "" {
		t.Fatalf("failure = %#v, want empty", updated.Status.Failure)
	}
}

func TestRunnerMarksBuildRunFailedOnCommandFailure(t *testing.T) {
	ctx := context.Background()
	repo := createGitRepository(t)
	buildRun := testBuildRun(repo)
	template := testPipelineTemplate([]cicdv1alpha1.PipelineStep{
		{
			Name:    "fail",
			Command: []string{"sh"},
			Args:    []string{"-c", "echo $SECRET_TOKEN && exit 9"},
			Env: []corev1.EnvVar{
				{Name: "SECRET_TOKEN", Value: "very-secret"},
			},
		},
	})
	k8sClient := newFakeRunnerClient(t, buildRun, testRepository(repo), template)
	cfg := testConfig(repo)
	runner := Runner{
		Client:    k8sClient,
		Git:       cloudivisiongit.ExecClient{},
		Steps:     steps.Runner{},
		Builder:   failBuilder{},
		Workspace: filepath.Join(t.TempDir(), "workspace"),
	}

	err := runner.Run(ctx, cfg)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if strings.Contains(err.Error(), "very-secret") {
		t.Fatalf("runner error leaked secret: %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if getErr := k8sClient.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); getErr != nil {
		t.Fatalf("get BuildRun error = %v", getErr)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed {
		t.Fatalf("phase = %q, want Failed", updated.Status.Phase)
	}
	if strings.Contains(updated.Status.Failure.Message, "very-secret") {
		t.Fatalf("status failure leaked secret: %#v", updated.Status.Failure)
	}
}

func TestRunnerRedactsRepositoryURLCredentialsOnCloneFailure(t *testing.T) {
	ctx := context.Background()
	repo := "https://user:very-secret-token@example.com/repo.git"
	buildRun := testBuildRun(repo)
	template := testPipelineTemplate(nil)
	k8sClient := newFakeRunnerClient(t, buildRun, testRepository(repo), template)
	cfg := testConfig(repo)
	runner := Runner{
		Client:    k8sClient,
		Git:       failingGit{message: "clone failed for " + repo},
		Steps:     steps.Runner{},
		Builder:   failBuilder{},
		Workspace: filepath.Join(t.TempDir(), "workspace"),
	}

	err := runner.Run(ctx, cfg)
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if strings.Contains(err.Error(), "very-secret-token") {
		t.Fatalf("runner error leaked repository token: %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if getErr := k8sClient.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); getErr != nil {
		t.Fatalf("get BuildRun error = %v", getErr)
	}
	if strings.Contains(updated.Status.Failure.Message, "very-secret-token") {
		t.Fatalf("status failure leaked repository token: %#v", updated.Status.Failure)
	}
}

func TestRunnerRecordsSupplyChainHookResults(t *testing.T) {
	ctx := context.Background()
	repo := createGitRepository(t)
	buildRun := testBuildRun(repo)
	template := testPipelineTemplate(nil)
	template.Spec.Build.Enabled = true
	template.Spec.Build.ContextDir = "."
	template.Spec.Build.Dockerfile = "Dockerfile"
	template.Spec.Build.Push = true
	template.Spec.SupplyChain = cicdv1alpha1.PipelineSupplyChainSpec{
		GenerateSBOM: true,
		ScanImage:    true,
		SignImage:    true,
	}
	k8sClient := newFakeRunnerClient(t, buildRun, testRepository(repo), template)
	cfg := testConfig(repo)
	runner := Runner{
		Client:     k8sClient,
		Git:        cloudivisiongit.ExecClient{},
		Steps:      steps.Runner{},
		Builder:    successBuilder{},
		SBOM:       fakeSBOMGenerator{},
		Scanner:    fakeScanner{},
		Signer:     fakeSigner{},
		Provenance: fakeProvenanceWriter{},
		Workspace:  filepath.Join(t.TempDir(), "workspace"),
	}

	if err := runner.Run(ctx, cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Image.Digest != "sha256:abc123" {
		t.Fatalf("image digest = %q, want sha256:abc123", updated.Status.Image.Digest)
	}
	if updated.Status.SupplyChain.SBOMPath != "sbom.spdx.json" {
		t.Fatalf("sbomPath = %q, want sbom.spdx.json", updated.Status.SupplyChain.SBOMPath)
	}
	if updated.Status.SupplyChain.SBOMDigest != "sha256:sbom" {
		t.Fatalf("sbomDigest = %q, want sha256:sbom", updated.Status.SupplyChain.SBOMDigest)
	}
	if updated.Status.SupplyChain.SignatureRef != "cosign://signature" {
		t.Fatalf("signatureRef = %q, want cosign://signature", updated.Status.SupplyChain.SignatureRef)
	}
	if updated.Status.SupplyChain.ScannerResultsRef != "scanner://result" {
		t.Fatalf("scannerResultsRef = %q, want scanner://result", updated.Status.SupplyChain.ScannerResultsRef)
	}
	if updated.Status.SupplyChain.ProvenanceRef != "oci://provenance" {
		t.Fatalf("provenanceRef = %q, want oci://provenance", updated.Status.SupplyChain.ProvenanceRef)
	}
	assertCondition(t, updated.Status.Conditions, ConditionSupplyChainReady)
}

type failBuilder struct{}

func (failBuilder) Build(context.Context, build.BuildRequest) (*build.BuildResult, error) {
	return nil, os.ErrInvalid
}

type successBuilder struct{}

func (successBuilder) Build(context.Context, build.BuildRequest) (*build.BuildResult, error) {
	return &build.BuildResult{
		ImageRepository: "ghcr.io/cloudivision/example",
		Tag:             "main",
		Digest:          "sha256:abc123",
	}, nil
}

type fakeSBOMGenerator struct{}

func (fakeSBOMGenerator) GenerateSBOM(context.Context, supplychain.SBOMRequest) (*supplychain.SBOMResult, error) {
	return &supplychain.SBOMResult{Path: "sbom.spdx.json", Digest: "sha256:sbom"}, nil
}

type fakeScanner struct{}

func (fakeScanner) ScanImage(context.Context, supplychain.ScanRequest) (*supplychain.ScanResult, error) {
	return &supplychain.ScanResult{ResultsRef: "scanner://result"}, nil
}

type fakeSigner struct{}

func (fakeSigner) SignImage(context.Context, supplychain.SignRequest) (*supplychain.SignResult, error) {
	return &supplychain.SignResult{SignatureRef: "cosign://signature"}, nil
}

type fakeProvenanceWriter struct{}

func (fakeProvenanceWriter) WriteProvenance(context.Context, supplychain.ProvenanceRequest) (*supplychain.ProvenanceResult, error) {
	return &supplychain.ProvenanceResult{Ref: "oci://provenance"}, nil
}

type failingGit struct {
	message string
}

func (g failingGit) Clone(context.Context, string, string) error {
	return fmt.Errorf("%s", g.message)
}

func (g failingGit) Checkout(context.Context, string, string) error {
	return nil
}

func newFakeRunnerClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add Kubernetes scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}).
		WithObjects(objects...).
		Build()
}

func createGitRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	run(t, dir, "git", "add", "README.md")
	run(t, dir, "git", "-c", "user.email=ci@example.com", "-c", "user.name=CI", "commit", "-m", "initial")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %s: %v", name, args, output, err)
	}
}

func testConfig(repo string) Config {
	return Config{
		BuildRunName:         "sample-buildrun",
		BuildRunNamespace:    "ci",
		ProjectName:          "sample-project",
		RepositoryURL:        repo,
		Revision:             "HEAD",
		Branch:               "main",
		PipelineTemplateName: "sample-template",
		ImageRepository:      "ghcr.io/cloudivision/example",
		ImageTag:             "main",
	}
}

func testBuildRun(repo string) *cicdv1alpha1.BuildRun {
	return &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-buildrun", Namespace: "ci"},
		Spec: cicdv1alpha1.BuildRunSpec{
			ProjectRef:          "sample-project",
			RepositoryRef:       "sample-repository",
			PipelineTemplateRef: "sample-template",
			Revision:            "HEAD",
			Branch:              "main",
			TriggeredBy: cicdv1alpha1.TriggeredBy{
				Type: cicdv1alpha1.TriggerTypeManual,
			},
			Image: cicdv1alpha1.ImageRef{
				Repository: "ghcr.io/cloudivision/example",
				Tag:        "main",
			},
			Params: map[string]string{
				"repo": repo,
			},
			Executor: cicdv1alpha1.ExecutorTypeJob,
		},
	}
}

func testRepository(repo string) *cicdv1alpha1.Repository {
	return &cicdv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-repository", Namespace: "ci"},
		Spec: cicdv1alpha1.RepositorySpec{
			ProjectRef:          "sample-project",
			Provider:            cicdv1alpha1.RepositoryProviderGeneric,
			URL:                 repo,
			DefaultBranch:       "main",
			PipelineTemplateRef: "sample-template",
		},
	}
}

func testPipelineTemplate(pipelineSteps []cicdv1alpha1.PipelineStep) *cicdv1alpha1.PipelineTemplate {
	return &cicdv1alpha1.PipelineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-template", Namespace: "ci"},
		Spec: cicdv1alpha1.PipelineTemplateSpec{
			Steps: pipelineSteps,
			Build: cicdv1alpha1.PipelineBuildSpec{
				Enabled: false,
			},
		},
	}
}

func assertCondition(t *testing.T, conditions []metav1.Condition, conditionType string) {
	t.Helper()
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == metav1.ConditionTrue {
			return
		}
	}
	t.Fatalf("condition %q not found in %#v", conditionType, conditions)
}

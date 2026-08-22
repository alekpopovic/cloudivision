package supplychain

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

func TestRequiredAdaptersReportMissingBinaries(t *testing.T) {
	ctx := context.Background()
	image := ImageContext{SourceDir: t.TempDir(), Image: cicdv1alpha1.ImageRef{Repository: "example/app", Tag: "v1"}}
	_, err := (SyftSBOMGenerator{Binary: "cloudivision-test-missing-syft"}).GenerateSBOM(ctx, SBOMRequest{ImageContext: image})
	if err == nil || !strings.Contains(err.Error(), "SBOM generation required") || !strings.Contains(err.Error(), "binary is unavailable") {
		t.Fatalf("GenerateSBOM() error = %v, want clear missing binary error", err)
	}
	_, err = (GrypeScanner{Binary: "cloudivision-test-missing-grype"}).ScanImage(ctx, ScanRequest{ImageContext: image})
	if err == nil || !strings.Contains(err.Error(), "image scan required") || !strings.Contains(err.Error(), "binary is unavailable") {
		t.Fatalf("ScanImage() error = %v, want clear missing binary error", err)
	}
	_, err = (CosignSigner{Binary: "cloudivision-test-missing-cosign", Keyless: true}).SignImage(ctx, SignRequest{ImageContext: image})
	if err == nil || !strings.Contains(err.Error(), "image signing required") || !strings.Contains(err.Error(), "binary is unavailable") {
		t.Fatalf("SignImage() error = %v, want clear missing binary error", err)
	}
}

func TestCosignRequiresExplicitSigningMode(t *testing.T) {
	_, err := (CosignSigner{Binary: "/bin/true"}).SignImage(context.Background(), SignRequest{ImageContext: ImageContext{Image: cicdv1alpha1.ImageRef{Repository: "example/app", Digest: "sha256:abc"}}})
	if err == nil || !strings.Contains(err.Error(), "explicit keyless mode or a mounted key Secret") {
		t.Fatalf("SignImage() error = %v", err)
	}
}

func TestReadGrypeSummaryCountsSeverities(t *testing.T) {
	path := filepath.Join(t.TempDir(), "grype.json")
	report := `{"matches":[{"vulnerability":{"severity":"Critical"}},{"vulnerability":{"severity":"High"}},{"vulnerability":{"severity":"Medium"}},{"vulnerability":{"severity":"Low"}},{"vulnerability":{"severity":"Critical"}}]}`
	if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := readGrypeSummary(path)
	if err != nil {
		t.Fatalf("readGrypeSummary() error = %v", err)
	}
	if result.Critical != 2 || result.High != 1 || result.Medium != 1 || result.Low != 1 {
		t.Fatalf("summary = %#v", result)
	}
}

func TestJSONProvenanceWriterWritesBuildInputs(t *testing.T) {
	started := time.Date(2026, 8, 22, 1, 2, 3, 0, time.UTC)
	completed := started.Add(time.Minute)
	result, err := (JSONProvenanceWriter{}).WriteProvenance(context.Background(), ProvenanceRequest{ImageContext: ImageContext{
		BuildRunName: "build-1", Namespace: "ci", SourceDir: t.TempDir(), RepositoryURL: "https://example.com/repo.git",
		CommitSHA: "abc123", PipelineTemplate: "go-build", Image: cicdv1alpha1.ImageRef{Repository: "example/app", Digest: "sha256:image"},
		StartedAt: &started, CompletedAt: &completed,
	}})
	if err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}
	data, err := os.ReadFile(result.Ref)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["buildRun"] != "build-1" || document["commitSHA"] != "abc123" || document["imageDigest"] != "sha256:image" {
		t.Fatalf("provenance = %#v", document)
	}
	info, err := os.Stat(result.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("provenance mode = %o, want 600", info.Mode().Perm())
	}
}

func TestImageReferencePrefersDigest(t *testing.T) {
	ref := imageReference(cicdv1alpha1.ImageRef{Repository: "example/app", Tag: "mutable", Digest: "sha256:immutable"})
	if !strings.HasSuffix(ref, "@sha256:immutable") {
		t.Fatalf("imageReference() = %q", ref)
	}
}

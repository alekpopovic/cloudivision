package supplychain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

type SyftSBOMGenerator struct{ Binary string }

func (a SyftSBOMGenerator) GenerateSBOM(ctx context.Context, req SBOMRequest) (*SBOMResult, error) {
	binary, err := requiredBinary(a.Binary, "syft")
	if err != nil {
		return nil, fmt.Errorf("SBOM generation required: %w", err)
	}
	path, err := artifactPath(req.SourceDir, "sbom.spdx.json")
	if err != nil {
		return nil, err
	}
	if err := runCommand(ctx, binary, imageReference(req.Image), "-o", "spdx-json="+path); err != nil {
		return nil, fmt.Errorf("run syft: %w", err)
	}
	digest, err := fileSHA256(path)
	if err != nil {
		return nil, fmt.Errorf("digest SBOM: %w", err)
	}
	return &SBOMResult{Path: path, Digest: digest}, nil
}

type GrypeScanner struct{ Binary string }

func (a GrypeScanner) ScanImage(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	binary, err := requiredBinary(a.Binary, "grype")
	if err != nil {
		return nil, fmt.Errorf("image scan required: %w", err)
	}
	path, err := artifactPath(req.SourceDir, "grype-results.json")
	if err != nil {
		return nil, err
	}
	if err := runCommand(ctx, binary, imageReference(req.Image), "-o", "json", "--file", path); err != nil {
		return nil, fmt.Errorf("run grype: %w", err)
	}
	result, err := readGrypeSummary(path)
	if err != nil {
		return nil, err
	}
	result.ResultsRef = path
	return result, nil
}

type CosignSigner struct {
	Binary  string
	Keyless bool
	KeyPath string
}

func (a CosignSigner) SignImage(ctx context.Context, req SignRequest) (*SignResult, error) {
	binary, err := requiredBinary(a.Binary, "cosign")
	if err != nil {
		return nil, fmt.Errorf("image signing required: %w", err)
	}
	if a.Keyless && a.KeyPath != "" {
		return nil, errors.New("cosign keyless and key-based modes are mutually exclusive")
	}
	args := []string{"sign", "--yes"}
	mode := ""
	switch {
	case a.Keyless:
		mode = "keyless"
	case a.KeyPath != "":
		mode = "key"
		args = append(args, "--key", a.KeyPath)
	default:
		return nil, errors.New("cosign signing requires explicit keyless mode or a mounted key Secret")
	}
	image := imageReference(req.Image)
	args = append(args, image)
	if err := runCommand(ctx, binary, args...); err != nil {
		return nil, fmt.Errorf("run cosign in %s mode: %w", mode, err)
	}
	return &SignResult{SignatureRef: "cosign://" + image}, nil
}

type JSONProvenanceWriter struct{}

type provenanceDocument struct {
	BuildRun          string     `json:"buildRun"`
	Namespace         string     `json:"namespace"`
	RepositoryURL     string     `json:"repositoryURL"`
	CommitSHA         string     `json:"commitSHA"`
	PipelineTemplate  string     `json:"pipelineTemplate"`
	Image             string     `json:"image"`
	ImageDigest       string     `json:"imageDigest"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	SBOMPath          string     `json:"sbomPath,omitempty"`
	SBOMDigest        string     `json:"sbomDigest,omitempty"`
	SignatureRef      string     `json:"signatureRef,omitempty"`
	ScannerResultsRef string     `json:"scannerResultsRef,omitempty"`
}

func (JSONProvenanceWriter) WriteProvenance(_ context.Context, req ProvenanceRequest) (*ProvenanceResult, error) {
	path, err := artifactPath(req.SourceDir, "provenance.json")
	if err != nil {
		return nil, err
	}
	document := provenanceDocument{
		BuildRun: req.BuildRunName, Namespace: req.Namespace, RepositoryURL: req.RepositoryURL,
		CommitSHA: req.CommitSHA, PipelineTemplate: req.PipelineTemplate, Image: imageReference(req.Image),
		ImageDigest: req.Image.Digest, StartedAt: req.StartedAt, CompletedAt: req.CompletedAt,
		SBOMPath: req.SBOMPath, SBOMDigest: req.SBOMDigest, SignatureRef: req.SignatureRef,
		ScannerResultsRef: req.ScannerResultsRef,
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode provenance: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, fmt.Errorf("write provenance: %w", err)
	}
	return &ProvenanceResult{Ref: path}, nil
}

func requiredBinary(configured, fallback string) (string, error) {
	binary := configured
	if binary == "" {
		binary = fallback
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("%s binary is unavailable: %w", fallback, err)
	}
	return path, nil
}

func runCommand(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func artifactPath(sourceDir, name string) (string, error) {
	if sourceDir == "" {
		return "", errors.New("source directory is required for supply-chain artifacts")
	}
	dir := filepath.Join(sourceDir, ".cloudivision")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create supply-chain artifact directory: %w", err)
	}
	return filepath.Join(dir, name), nil
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func imageReference(image cicdv1alpha1.ImageRef) string {
	ref := image.Repository
	if image.Digest != "" {
		return ref + "@" + image.Digest
	}
	if image.Tag != "" {
		return ref + ":" + image.Tag
	}
	return ref
}

func readGrypeSummary(path string) (*ScanResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read grype results: %w", err)
	}
	var report struct {
		Matches []struct {
			Vulnerability struct {
				Severity string `json:"severity"`
			} `json:"vulnerability"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse grype results: %w", err)
	}
	result := &ScanResult{}
	for _, match := range report.Matches {
		switch strings.ToLower(match.Vulnerability.Severity) {
		case "critical":
			result.Critical++
		case "high":
			result.High++
		case "medium":
			result.Medium++
		case "low", "negligible":
			result.Low++
		}
	}
	return result, nil
}

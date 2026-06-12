package supplychain

import (
	"context"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

type ImageContext struct {
	BuildRunName string
	Namespace    string
	ProjectName  string
	SourceDir    string
	Image        cicdv1alpha1.ImageRef
}

type SBOMRequest struct {
	ImageContext
}

type SBOMResult struct {
	Path   string
	Digest string
}

type ScanRequest struct {
	ImageContext
	SBOMPath string
}

type ScanResult struct {
	ResultsRef string
}

type SignRequest struct {
	ImageContext
}

type SignResult struct {
	SignatureRef string
}

type ProvenanceRequest struct {
	ImageContext
	SBOMPath          string
	SBOMDigest        string
	SignatureRef      string
	ScannerResultsRef string
}

type ProvenanceResult struct {
	Ref string
}

type SBOMGenerator interface {
	GenerateSBOM(ctx context.Context, req SBOMRequest) (*SBOMResult, error)
}

type VulnerabilityScanner interface {
	ScanImage(ctx context.Context, req ScanRequest) (*ScanResult, error)
}

type ImageSigner interface {
	SignImage(ctx context.Context, req SignRequest) (*SignResult, error)
}

type ProvenanceWriter interface {
	WriteProvenance(ctx context.Context, req ProvenanceRequest) (*ProvenanceResult, error)
}

type NoopSBOMGenerator struct{}

func (NoopSBOMGenerator) GenerateSBOM(context.Context, SBOMRequest) (*SBOMResult, error) {
	return &SBOMResult{}, nil
}

type NoopScanner struct{}

func (NoopScanner) ScanImage(context.Context, ScanRequest) (*ScanResult, error) {
	return &ScanResult{}, nil
}

type NoopSigner struct{}

func (NoopSigner) SignImage(context.Context, SignRequest) (*SignResult, error) {
	return &SignResult{}, nil
}

type NoopProvenanceWriter struct{}

func (NoopProvenanceWriter) WriteProvenance(context.Context, ProvenanceRequest) (*ProvenanceResult, error) {
	return &ProvenanceResult{}, nil
}

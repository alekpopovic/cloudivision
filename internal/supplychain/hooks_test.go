package supplychain

import (
	"context"
	"testing"
)

func TestNoopHooksReturnEmptyResults(t *testing.T) {
	ctx := context.Background()
	sbom, err := NoopSBOMGenerator{}.GenerateSBOM(ctx, SBOMRequest{})
	if err != nil {
		t.Fatalf("GenerateSBOM() error = %v", err)
	}
	if sbom.Path != "" || sbom.Digest != "" {
		t.Fatalf("SBOM result = %#v, want empty", sbom)
	}
	scan, err := NoopScanner{}.ScanImage(ctx, ScanRequest{})
	if err != nil {
		t.Fatalf("ScanImage() error = %v", err)
	}
	if scan.ResultsRef != "" {
		t.Fatalf("scan result = %#v, want empty", scan)
	}
	signature, err := NoopSigner{}.SignImage(ctx, SignRequest{})
	if err != nil {
		t.Fatalf("SignImage() error = %v", err)
	}
	if signature.SignatureRef != "" {
		t.Fatalf("signature result = %#v, want empty", signature)
	}
	provenance, err := NoopProvenanceWriter{}.WriteProvenance(ctx, ProvenanceRequest{})
	if err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}
	if provenance.Ref != "" {
		t.Fatalf("provenance result = %#v, want empty", provenance)
	}
}

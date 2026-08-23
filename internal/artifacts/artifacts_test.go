package artifacts

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestMemoryStoreLifecycle(t *testing.T) {
	store := NewMemoryStore()
	data := []byte("report")
	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	ref, err := store.Put(context.Background(), PutArtifactRequest{Namespace: "ci", BuildRun: "build-1", Name: "report.xml", Path: "reports/report.xml", Type: "application/xml", Digest: digest, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.Get(context.Background(), *ref)
	if err != nil || string(artifact.Data) != "report" || artifact.Ref.Digest != digest {
		t.Fatalf("artifact=%#v err=%v", artifact, err)
	}
	refs, err := store.List(context.Background(), ListArtifactsRequest{Namespace: "ci", BuildRun: "build-1"})
	if err != nil || len(refs) != 1 {
		t.Fatalf("refs=%#v err=%v", refs, err)
	}
	if err := store.Delete(context.Background(), *ref); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), *ref); err != ErrNotFound {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestLocalStorePersistsArtifactAndMetadata(t *testing.T) {
	store := LocalStore{Root: t.TempDir()}
	ref, err := store.Put(context.Background(), PutArtifactRequest{Namespace: "ci", BuildRun: "build-1", Name: "binary", Path: "dist/binary", Type: "application/octet-stream", Digest: "sha256:abc", Data: []byte("binary")})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := store.Get(context.Background(), *ref)
	if err != nil || string(artifact.Data) != "binary" {
		t.Fatalf("artifact=%#v err=%v", artifact, err)
	}
	refs, err := store.List(context.Background(), ListArtifactsRequest{Namespace: "ci", BuildRun: "build-1"})
	if err != nil || len(refs) != 1 || refs[0].Path != "dist/binary" {
		t.Fatalf("refs=%#v err=%v", refs, err)
	}
}

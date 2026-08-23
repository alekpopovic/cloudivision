package artifacts

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("artifact not found")
	ErrDisabled       = errors.New("artifact storage is disabled")
	ErrNotImplemented = errors.New("artifact backend is not implemented")
)

type ArtifactRef struct {
	Namespace string    `json:"namespace"`
	BuildRun  string    `json:"buildRun"`
	Name      string    `json:"name"`
	Path      string    `json:"path,omitempty"`
	Type      string    `json:"type,omitempty"`
	Size      int64     `json:"size"`
	Digest    string    `json:"digest"`
	Ref       string    `json:"ref"`
	CreatedAt time.Time `json:"createdAt"`
}

type Artifact struct {
	Ref  ArtifactRef
	Data []byte
}

type PutArtifactRequest struct {
	Namespace     string
	BuildRun      string
	Name          string
	Path          string
	Type          string
	Digest        string
	Data          []byte
	RetentionDays int
}

type ListArtifactsRequest struct {
	Namespace string
	BuildRun  string
}

type ArtifactStore interface {
	Put(ctx context.Context, req PutArtifactRequest) (*ArtifactRef, error)
	Get(ctx context.Context, ref ArtifactRef) (*Artifact, error)
	List(ctx context.Context, req ListArtifactsRequest) ([]ArtifactRef, error)
	Delete(ctx context.Context, ref ArtifactRef) error
}

type NoopStore struct{}

func (NoopStore) Put(context.Context, PutArtifactRequest) (*ArtifactRef, error) {
	return nil, ErrDisabled
}
func (NoopStore) Get(context.Context, ArtifactRef) (*Artifact, error) { return nil, ErrDisabled }
func (NoopStore) List(context.Context, ListArtifactsRequest) ([]ArtifactRef, error) {
	return []ArtifactRef{}, nil
}
func (NoopStore) Delete(context.Context, ArtifactRef) error { return nil }

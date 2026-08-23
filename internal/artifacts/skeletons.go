package artifacts

import "context"

type ObjectStore struct{}
type OCIStore struct{}

func (ObjectStore) Put(context.Context, PutArtifactRequest) (*ArtifactRef, error) {
	return nil, ErrNotImplemented
}
func (ObjectStore) Get(context.Context, ArtifactRef) (*Artifact, error) {
	return nil, ErrNotImplemented
}
func (ObjectStore) List(context.Context, ListArtifactsRequest) ([]ArtifactRef, error) {
	return nil, ErrNotImplemented
}
func (ObjectStore) Delete(context.Context, ArtifactRef) error { return ErrNotImplemented }

func (OCIStore) Put(context.Context, PutArtifactRequest) (*ArtifactRef, error) {
	return nil, ErrNotImplemented
}
func (OCIStore) Get(context.Context, ArtifactRef) (*Artifact, error) { return nil, ErrNotImplemented }
func (OCIStore) List(context.Context, ListArtifactsRequest) ([]ArtifactRef, error) {
	return nil, ErrNotImplemented
}
func (OCIStore) Delete(context.Context, ArtifactRef) error { return ErrNotImplemented }

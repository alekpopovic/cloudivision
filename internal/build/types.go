package build

import "context"

type CacheMode string

const (
	CacheModeInline   CacheMode = "inline"
	CacheModeRegistry CacheMode = "registry"
	CacheModeLocal    CacheMode = "local"
)

type CacheConfig struct {
	Enabled bool
	Mode    CacheMode
	Ref     string
}

type ImageBuilder interface {
	Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
}

type BuildRequest struct {
	ContextDir      string
	Dockerfile      string
	ImageRepository string
	ImageTag        string
	Push            bool
	BuildArgs       map[string]string
	Target          string
	Platforms       []string
	Labels          map[string]string
	Cache           CacheConfig
	Env             map[string]string
}

type BuildResult struct {
	ImageRepository string
	Tag             string
	Digest          string
	SBOMPath        string
}

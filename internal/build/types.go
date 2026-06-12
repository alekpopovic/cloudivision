package build

import "context"

type ImageBuilder interface {
	Build(ctx context.Context, req BuildRequest) (*BuildResult, error)
}

type BuildRequest struct {
	ContextDir      string
	Dockerfile      string
	ImageRepository string
	ImageTag        string
	Push            bool
	Env             map[string]string
}

type BuildResult struct {
	ImageRepository string
	Tag             string
	Digest          string
	SBOMPath        string
}

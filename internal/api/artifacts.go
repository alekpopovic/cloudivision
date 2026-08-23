package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/artifacts"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s Server) buildRunArtifacts(w http.ResponseWriter, r *http.Request) {
	buildRun, ok := s.artifactBuildRun(w, r)
	if !ok {
		return
	}
	items := buildRun.Status.Artifacts
	if items == nil {
		items = []cicdv1alpha1.BuildRunArtifactStatus{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) buildRunArtifact(w http.ResponseWriter, r *http.Request) {
	if s.ArtifactStore == nil {
		s.writeError(w, notFound("artifact storage is disabled"))
		return
	}
	buildRun, ok := s.artifactBuildRun(w, r)
	if !ok {
		return
	}
	name := r.PathValue("artifactName")
	var status *cicdv1alpha1.BuildRunArtifactStatus
	for i := range buildRun.Status.Artifacts {
		if buildRun.Status.Artifacts[i].Name == name {
			status = &buildRun.Status.Artifacts[i]
			break
		}
	}
	if status == nil {
		s.writeError(w, notFound("artifact not found"))
		return
	}
	ref := artifacts.ArtifactRef{Namespace: buildRun.Namespace, BuildRun: buildRun.Name, Name: status.Name, Path: status.Path, Type: status.Type, Size: status.Size, Digest: status.Digest, Ref: status.Ref}
	artifact, err := s.ArtifactStore.Get(r.Context(), ref)
	if err != nil {
		if errors.Is(err, artifacts.ErrNotFound) || errors.Is(err, artifacts.ErrDisabled) {
			s.writeError(w, notFound("artifact content not found"))
		} else {
			s.writeError(w, fmt.Errorf("read artifact: %w", err))
		}
		return
	}
	contentType := status.Type
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(artifact.Data)))
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(status.Name, `"`, "")+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(artifact.Data)
}

func (s Server) artifactBuildRun(w http.ResponseWriter, r *http.Request) (*cicdv1alpha1.BuildRun, bool) {
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: r.PathValue("namespace")}, buildRun); err != nil {
		s.writeError(w, err)
		return nil, false
	}
	return buildRun, true
}

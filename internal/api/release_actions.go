package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s Server) promoteRelease(w http.ResponseWriter, r *http.Request) {
	var request ReleasePromoteRequest
	if !s.decode(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.TargetEnvironmentRef) == "" {
		s.writeError(w, badRequest("targetEnvironmentRef is required"))
		return
	}
	source, ok := s.getReleaseForAction(w, r)
	if !ok {
		return
	}
	if source.Status.Phase != cicdv1alpha1.ReleasePhaseDeployed {
		s.writeError(w, conflict("only a deployed release can be promoted"))
		return
	}
	target := &cicdv1alpha1.Environment{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: source.Namespace, Name: request.TargetEnvironmentRef}, target); err != nil {
		s.writeError(w, err)
		return
	}
	if target.Spec.ProjectRef != source.Spec.ProjectRef {
		s.writeError(w, badRequest("target environment does not belong to the release project"))
		return
	}
	image, buildRunRef, err := s.resolvedReleaseImage(r.Context(), source)
	if err != nil {
		s.writeError(w, err)
		return
	}
	created := newReleaseAction(source, target.Name, buildRunRef, image)
	created.Name = releaseActionName(source.Name, "promote", target.Name)
	created.Spec.PromotedFrom = source.Name
	created.Spec.Approval.Required = target.Spec.RequiresApproval || target.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction
	if err := s.Client.Create(r.Context(), created); err != nil {
		s.writeError(w, err)
		return
	}
	actor := auth.ActorFromContext(r.Context(), request.Actor)
	s.recordAudit(r.Context(), audit.Event{Type: "ReleasePromoted", Actor: actor, Project: created.Spec.ProjectRef, BuildRun: created.Spec.BuildRunRef, Release: created.Name, Message: "Release promotion created.", Metadata: auditMetadata(map[string]string{"sourceRelease": source.Name, "targetEnvironment": target.Name, "namespace": source.Namespace})})
	writeJSON(w, http.StatusCreated, releaseDTO(*created))
}

func (s Server) rollbackRelease(w http.ResponseWriter, r *http.Request) {
	var request ReleaseRollbackRequest
	if !s.decode(w, r, &request) {
		return
	}
	if strings.TrimSpace(request.TargetReleaseRef) == "" {
		s.writeError(w, badRequest("targetReleaseRef is required"))
		return
	}
	source, ok := s.getReleaseForAction(w, r)
	if !ok {
		return
	}
	targetRelease := &cicdv1alpha1.Release{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: source.Namespace, Name: request.TargetReleaseRef}, targetRelease); err != nil {
		s.writeError(w, err)
		return
	}
	if targetRelease.Status.Phase != cicdv1alpha1.ReleasePhaseDeployed {
		s.writeError(w, conflict("rollback target must be a previously deployed release"))
		return
	}
	if targetRelease.Spec.ProjectRef != source.Spec.ProjectRef {
		s.writeError(w, badRequest("rollback target does not belong to the release project"))
		return
	}
	environment := &cicdv1alpha1.Environment{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: source.Namespace, Name: source.Spec.EnvironmentRef}, environment); err != nil {
		s.writeError(w, err)
		return
	}
	image, buildRunRef, err := s.resolvedReleaseImage(r.Context(), targetRelease)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if environment.Spec.Policy.RequireImageDigest && image.Digest == "" {
		s.writeError(w, conflict("rollback blocked by policy: target image has no digest"))
		return
	}
	if environment.Spec.Policy.RequireSignedImages {
		buildRun := &cicdv1alpha1.BuildRun{}
		if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: source.Namespace, Name: buildRunRef}, buildRun); err != nil {
			s.writeError(w, err)
			return
		}
		if buildRun.Status.SupplyChain.SignatureRef == "" {
			s.writeError(w, conflict("rollback blocked by policy: target image is not signed"))
			return
		}
	}
	created := newReleaseAction(source, source.Spec.EnvironmentRef, buildRunRef, image)
	created.Name = releaseActionName(source.Name, "rollback", targetRelease.Name)
	created.Spec.RollbackOf = source.Name
	created.Spec.RollbackTo = targetRelease.Name
	created.Spec.Approval.Required = environment.Spec.RequiresApproval || environment.Spec.Type == cicdv1alpha1.EnvironmentTypeProduction
	created.Annotations = mergeStringMap(created.Annotations, map[string]string{"cloudivision.io/rollback-reason": request.Reason})
	if err := s.Client.Create(r.Context(), created); err != nil {
		s.writeError(w, err)
		return
	}
	actor := auth.ActorFromContext(r.Context(), request.Actor)
	s.recordAudit(r.Context(), audit.Event{Type: "ReleaseRollbackCreated", Actor: actor, Project: created.Spec.ProjectRef, BuildRun: created.Spec.BuildRunRef, Release: created.Name, Message: "Release rollback created.", Metadata: auditMetadata(map[string]string{"rollbackOf": source.Name, "rollbackTo": targetRelease.Name, "reason": request.Reason, "namespace": source.Namespace})})
	writeJSON(w, http.StatusCreated, releaseDTO(*created))
}

func (s Server) getReleaseForAction(w http.ResponseWriter, r *http.Request) (*cicdv1alpha1.Release, bool) {
	release := &cicdv1alpha1.Release{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: r.PathValue("namespace"), Name: r.PathValue("name")}, release); err != nil {
		s.writeError(w, err)
		return nil, false
	}
	return release, true
}

func (s Server) resolvedReleaseImage(ctx context.Context, release *cicdv1alpha1.Release) (cicdv1alpha1.ImageRef, string, error) {
	image := release.Spec.Image
	if image.Digest != "" {
		return image, release.Spec.BuildRunRef, nil
	}
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := s.Client.Get(ctx, client.ObjectKey{Namespace: release.Namespace, Name: release.Spec.BuildRunRef}, buildRun); err != nil {
		if image.Repository != "" {
			return image, release.Spec.BuildRunRef, nil
		}
		return image, release.Spec.BuildRunRef, fmt.Errorf("resolve release BuildRun image: %w", err)
	}
	if buildRun.Status.Image != nil && buildRun.Status.Image.Digest != "" {
		image = *buildRun.Status.Image
	}
	return image, release.Spec.BuildRunRef, nil
}

func newReleaseAction(source *cicdv1alpha1.Release, environment, buildRun string, image cicdv1alpha1.ImageRef) *cicdv1alpha1.Release {
	return &cicdv1alpha1.Release{ObjectMeta: metav1.ObjectMeta{Namespace: source.Namespace, Labels: map[string]string{"cicd.cloudivision.io/project": source.Spec.ProjectRef}}, Spec: cicdv1alpha1.ReleaseSpec{ProjectRef: source.Spec.ProjectRef, EnvironmentRef: environment, BuildRunRef: buildRun, Image: image, Strategy: cicdv1alpha1.ReleaseStrategyGitOps, PromotionMode: source.Spec.PromotionMode, PullRequest: source.Spec.PullRequest, DeploymentTimeout: source.Spec.DeploymentTimeout}}
}

func releaseActionName(source, action, target string) string {
	suffix := strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	prefix := strings.Trim(strings.ToLower(source+"-"+action+"-"+target), "-")
	max := 63 - len(suffix) - 1
	if len(prefix) > max {
		prefix = strings.TrimRight(prefix[:max], "-")
	}
	return prefix + "-" + suffix
}

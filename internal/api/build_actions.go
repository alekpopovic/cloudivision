package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/auth"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	retryOfAnnotation = "cloudivision.io/retry-of"
	rerunOfAnnotation = "cloudivision.io/rerun-of"
)

func (s Server) cancelBuildRun(w http.ResponseWriter, r *http.Request) {
	buildRun, ok := s.loadBuildRunAction(w, r)
	if !ok {
		return
	}
	if buildRun.Status.Phase == cicdv1alpha1.BuildRunPhaseCancelled {
		writeJSON(w, http.StatusOK, buildRunDTO(*buildRun))
		return
	}
	switch buildRun.Status.Phase {
	case "", cicdv1alpha1.BuildRunPhasePending, cicdv1alpha1.BuildRunPhaseQueued, cicdv1alpha1.BuildRunPhaseRunning:
	default:
		s.writeError(w, conflict("only Pending, Queued, or Running BuildRuns can be cancelled"))
		return
	}
	if err := s.deleteBuildRunJobs(r, buildRun); err != nil {
		s.writeError(w, err)
		return
	}
	now := metav1.Now()
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseCancelled
	buildRun.Status.CompletedAt = &now
	apiMeta.SetStatusCondition(&buildRun.Status.Conditions, metav1.Condition{
		Type: "Cancelled", Status: metav1.ConditionTrue, Reason: "CancelledByUser",
		Message: "BuildRun was cancelled through the API.", LastTransitionTime: now,
	})
	if err := s.Client.Status().Update(r.Context(), buildRun); err != nil {
		s.writeError(w, fmt.Errorf("update cancelled BuildRun status: %w", err))
		return
	}
	s.recordAudit(r.Context(), audit.Event{Type: "BuildRunCancelled", Actor: auth.ActorFromContext(r.Context(), ""), Project: buildRun.Spec.ProjectRef, BuildRun: buildRun.Name, Message: "BuildRun cancelled and runner Job stopped."})
	writeJSON(w, http.StatusOK, buildRunDTO(*buildRun))
}

func (s Server) retryBuildRun(w http.ResponseWriter, r *http.Request) {
	s.cloneBuildRun(w, r, "retry")
}

func (s Server) rerunBuildRun(w http.ResponseWriter, r *http.Request) {
	s.cloneBuildRun(w, r, "rerun")
}

func (s Server) cloneBuildRun(w http.ResponseWriter, r *http.Request, action string) {
	source, ok := s.loadBuildRunAction(w, r)
	if !ok {
		return
	}
	if action == "retry" && source.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed && source.Status.Phase != cicdv1alpha1.BuildRunPhaseCancelled {
		s.writeError(w, conflict("retry is allowed only for Failed or Cancelled BuildRuns"))
		return
	}
	if action == "rerun" && source.Status.Phase != cicdv1alpha1.BuildRunPhaseSucceeded && source.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed && source.Status.Phase != cicdv1alpha1.BuildRunPhaseCancelled {
		s.writeError(w, conflict("rerun is allowed only for terminal BuildRuns"))
		return
	}
	annotation := rerunOfAnnotation
	if action == "retry" {
		annotation = retryOfAnnotation
	}
	suffix := strconv.FormatInt(time.Now().UTC().UnixNano(), 36)
	name := strings.TrimSuffix(fmt.Sprintf("%s-%s-%s", source.Name, action, suffix), "-")
	if len(name) > 63 {
		name = strings.TrimSuffix(name[:63], "-")
	}
	clone := &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: source.Namespace, Annotations: map[string]string{annotation: source.Name}},
		Spec:       *source.Spec.DeepCopy(),
	}
	clone.Spec.TriggeredBy = cicdv1alpha1.TriggeredBy{Type: cicdv1alpha1.TriggerTypeManual, Actor: auth.ActorFromContext(r.Context(), action)}
	if err := s.Client.Create(r.Context(), clone); err != nil {
		s.writeError(w, err)
		return
	}
	eventType := "BuildRunRerunCreated"
	if action == "retry" {
		eventType = "BuildRunRetryCreated"
	}
	s.recordAudit(r.Context(), audit.Event{Type: eventType, Actor: auth.ActorFromContext(r.Context(), ""), Project: clone.Spec.ProjectRef, BuildRun: clone.Name, Message: fmt.Sprintf("Created %s of BuildRun %s.", action, source.Name), Metadata: auditMetadata(map[string]string{"sourceBuildRun": source.Name})})
	writeJSON(w, http.StatusCreated, buildRunDTO(*clone))
}

func (s Server) loadBuildRunAction(w http.ResponseWriter, r *http.Request) (*cicdv1alpha1.BuildRun, bool) {
	buildRun := &cicdv1alpha1.BuildRun{}
	if err := s.Client.Get(r.Context(), client.ObjectKey{Name: r.PathValue("name"), Namespace: r.PathValue("namespace")}, buildRun); err != nil {
		s.writeError(w, err)
		return nil, false
	}
	return buildRun, true
}

func (s Server) deleteBuildRunJobs(r *http.Request, buildRun *cicdv1alpha1.BuildRun) error {
	if buildRun.Status.JobRef.Name != "" {
		job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Name: buildRun.Status.JobRef.Name, Namespace: buildRun.Namespace}}
		if err := s.Client.Delete(r.Context(), job, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete runner Job %q: %w", job.Name, err)
		}
		return nil
	}
	var jobs batchv1.JobList
	selector := labels.SelectorFromSet(labels.Set{"cloudivision.io/buildrun": buildRun.Name})
	if err := s.Client.List(r.Context(), &jobs, client.InNamespace(buildRun.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return fmt.Errorf("list runner Jobs: %w", err)
	}
	for i := range jobs.Items {
		if err := s.Client.Delete(r.Context(), &jobs.Items[i], client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete runner Job %q: %w", jobs.Items[i].Name, err)
		}
	}
	return nil
}

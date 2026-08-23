package controller

import (
	"context"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	jobexecutor "github.com/cloudivision/cloudivision/internal/executor/job"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBuildRunReconcileCreatesOneJob(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("second Reconcile() error = %v", err)
	}

	jobs := &batchv1.JobList{}
	if err := reconciler.List(ctx, jobs, client.InNamespace(buildRun.Namespace)); err != nil {
		t.Fatalf("List Jobs error = %v", err)
	}
	if len(jobs.Items) != 1 {
		t.Fatalf("len(jobs.Items) = %d, want 1", len(jobs.Items))
	}

	job := jobs.Items[0]
	if job.Labels["cloudivision.io/buildrun"] != buildRun.Name {
		t.Fatalf("buildrun label = %q, want %q", job.Labels["cloudivision.io/buildrun"], buildRun.Name)
	}
	if len(job.OwnerReferences) != 1 || job.OwnerReferences[0].Name != buildRun.Name {
		t.Fatalf("ownerReferences = %#v, want BuildRun owner", job.OwnerReferences)
	}
	if got := envValue(job.Spec.Template.Spec.Containers[0].Env, "REPOSITORY_URL"); got != "https://github.com/cloudivision/example.git" {
		t.Fatalf("REPOSITORY_URL = %q", got)
	}
	assertSecureJobSpec(t, &job)
}

func TestProjectConcurrencyQueuesThenStartsBuildRun(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	setProjectQuotas(t, ctx, reconciler, &cicdv1alpha1.ProjectQuotaSpec{MaxConcurrentBuildRuns: 1, MaxQueuedBuildRuns: 2})
	active := createBuildRunWithPhase(t, ctx, reconciler, "active-build", cicdv1alpha1.BuildRunPhaseRunning)
	result, err := reconciler.Reconcile(ctx, requestFor(buildRun))
	if err != nil || result.RequeueAfter <= 0 {
		t.Fatalf("Reconcile() result=%#v err=%v", result, err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseQueued || !hasConditionReason(updated.Status.Conditions, "Queued", "ConcurrencyQuotaReached") {
		t.Fatalf("status = %#v", updated.Status)
	}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: jobexecutor.NameForBuildRun(buildRun.Name), Namespace: buildRun.Namespace}, &batchv1.Job{}); !apierrors.IsNotFound(err) {
		t.Fatalf("queued Job error = %v, want NotFound", err)
	}
	active.Status.Phase = cicdv1alpha1.BuildRunPhaseSucceeded
	if err := reconciler.Status().Update(ctx, active); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatal(err)
	}
	getRunnerJob(t, ctx, reconciler, buildRun)
}

func TestProjectQueueLimitFailsBuildRun(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	setProjectQuotas(t, ctx, reconciler, &cicdv1alpha1.ProjectQuotaSpec{MaxConcurrentBuildRuns: 1, MaxQueuedBuildRuns: 1})
	createBuildRunWithPhase(t, ctx, reconciler, "active-build", cicdv1alpha1.BuildRunPhaseRunning)
	createBuildRunWithPhase(t, ctx, reconciler, "queued-build", cicdv1alpha1.BuildRunPhaseQueued)
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatal(err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed || updated.Status.Failure.Reason != "QuotaExceeded" {
		t.Fatalf("status = %#v", updated.Status)
	}
}

func TestProjectQuotaAppliesStricterTimeout(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	setProjectQuotas(t, ctx, reconciler, &cicdv1alpha1.ProjectQuotaSpec{MaxBuildDurationSeconds: 60, MaxCPU: "250m", MaxMemory: "256Mi"})
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatal(err)
	}
	job := getRunnerJob(t, ctx, reconciler, buildRun)
	if job.Spec.ActiveDeadlineSeconds == nil || *job.Spec.ActiveDeadlineSeconds != 60 {
		t.Fatalf("activeDeadlineSeconds = %#v", job.Spec.ActiveDeadlineSeconds)
	}
	limits := job.Spec.Template.Spec.Containers[0].Resources.Limits
	if limits.Cpu().String() != "250m" || limits.Memory().String() != "256Mi" {
		t.Fatalf("limits = %#v", limits)
	}
}

func setProjectQuotas(t *testing.T, ctx context.Context, reconciler *BuildRunReconciler, quotas *cicdv1alpha1.ProjectQuotaSpec) {
	t.Helper()
	project := &cicdv1alpha1.Project{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: "sample-project", Namespace: "ci"}, project); err != nil {
		t.Fatal(err)
	}
	project.Spec.Quotas = quotas
	if err := reconciler.Update(ctx, project); err != nil {
		t.Fatal(err)
	}
}

func createBuildRunWithPhase(t *testing.T, ctx context.Context, reconciler *BuildRunReconciler, name string, phase cicdv1alpha1.BuildRunPhase) *cicdv1alpha1.BuildRun {
	t.Helper()
	buildRun := testBuildRun()
	buildRun.Name = name
	buildRun.ResourceVersion = ""
	buildRun.UID = ""
	buildRun.Status = cicdv1alpha1.BuildRunStatus{}
	if err := reconciler.Create(ctx, buildRun); err != nil {
		t.Fatal(err)
	}
	buildRun.Status.Phase = phase
	if err := reconciler.Status().Update(ctx, buildRun); err != nil {
		t.Fatal(err)
	}
	return buildRun
}

func TestBuildRunReconcileSetsPolicyDeniedBeforeCreatingJob(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	template := &cicdv1alpha1.PipelineTemplate{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: buildRun.Spec.PipelineTemplateRef, Namespace: buildRun.Namespace}, template); err != nil {
		t.Fatal(err)
	}
	template.Spec.Security.AllowPrivileged = true
	if err := reconciler.Update(ctx, template); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatal(err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: buildRun.Name, Namespace: buildRun.Namespace}, updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed || !hasConditionReason(updated.Status.Conditions, "PolicyDenied", "PolicyDenied") {
		t.Fatalf("status = %#v, want PolicyDenied", updated.Status)
	}
	if updated.Status.Policy.Allowed || len(updated.Status.Policy.Violations) != 1 {
		t.Fatalf("policy = %#v", updated.Status.Policy)
	}
	job := &batchv1.Job{}
	err := reconciler.Get(ctx, types.NamespacedName{Name: jobexecutor.NameForBuildRun(buildRun.Name), Namespace: buildRun.Namespace}, job)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("get Job error = %v, want not found", err)
	}
}

func TestBuildRunReconcileDoesNotAddUnneededFinalizer(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	for _, finalizer := range updated.Finalizers {
		if finalizer == legacyBuildRunFinalizer {
			t.Fatalf("unneeded legacy finalizer was added: %#v", updated.Finalizers)
		}
	}
}

func TestBuildRunDeletionRemovesLegacyFinalizer(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Finalizers = []string{legacyBuildRunFinalizer}
	if err := reconciler.Update(ctx, buildRun); err != nil {
		t.Fatalf("add legacy finalizer error = %v", err)
	}
	if err := reconciler.Delete(ctx, buildRun); err != nil {
		t.Fatalf("delete BuildRun error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() deletion error = %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated)
	if err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("get BuildRun after deletion error = %v", err)
	}
	if err == nil {
		for _, finalizer := range updated.Finalizers {
			if finalizer == legacyBuildRunFinalizer {
				t.Fatalf("legacy finalizer remains after deletion: %#v", updated.Finalizers)
			}
		}
	}
}

func TestBuildRunReconcileRecreatesDeletedJob(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	job := getRunnerJob(t, ctx, reconciler, buildRun)
	if err := reconciler.Delete(ctx, job); err != nil {
		t.Fatalf("delete Job error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() after Job deletion error = %v", err)
	}
	recreated := getRunnerJob(t, ctx, reconciler, buildRun)
	if len(recreated.OwnerReferences) != 1 || recreated.OwnerReferences[0].UID != buildRun.UID {
		t.Fatalf("recreated Job ownerReferences = %#v", recreated.OwnerReferences)
	}
}

func TestBuildRunReconcileMarksSucceeded(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	job := getRunnerJob(t, ctx, reconciler, buildRun)
	job.Status.Succeeded = 1
	if err := reconciler.Status().Update(ctx, job); err != nil {
		t.Fatalf("update Job status error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() after success error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseSucceeded {
		t.Fatalf("phase = %q, want Succeeded", updated.Status.Phase)
	}
	if updated.Status.CompletedAt == nil {
		t.Fatal("completedAt = nil, want timestamp")
	}
	if updated.Status.Image == nil || updated.Status.Image.Repository != buildRun.Spec.Image.Repository {
		t.Fatalf("status image repository = %q", updated.Status.Image.Repository)
	}
}

func TestBuildRunSuccessCreatesReleaseOnce(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	job := getRunnerJob(t, ctx, reconciler, buildRun)
	job.Status.Succeeded = 1
	if err := reconciler.Status().Update(ctx, job); err != nil {
		t.Fatalf("update Job status error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() after success error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("second Reconcile() after success error = %v", err)
	}

	releases := &cicdv1alpha1.ReleaseList{}
	if err := reconciler.List(ctx, releases, client.InNamespace(buildRun.Namespace)); err != nil {
		t.Fatalf("List Releases error = %v", err)
	}
	if len(releases.Items) != 1 {
		t.Fatalf("len(releases.Items) = %d, want 1", len(releases.Items))
	}
	release := releases.Items[0]
	if release.Name != "sample-buildrun-sample-environment" {
		t.Fatalf("release name = %q", release.Name)
	}
	if release.Spec.BuildRunRef != buildRun.Name {
		t.Fatalf("buildRunRef = %q, want %q", release.Spec.BuildRunRef, buildRun.Name)
	}
	if release.Spec.Strategy != cicdv1alpha1.ReleaseStrategyGitOps {
		t.Fatalf("strategy = %q, want gitops", release.Spec.Strategy)
	}
	if len(release.OwnerReferences) != 1 || release.OwnerReferences[0].Name != buildRun.Name {
		t.Fatalf("Release ownerReferences = %#v, want BuildRun owner", release.OwnerReferences)
	}
}

func TestTerminalBuildRunRecreatesDeletedRelease(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseSucceeded
	buildRun.Status.Image = &buildRun.Spec.Image
	if err := reconciler.Status().Update(ctx, buildRun); err != nil {
		t.Fatalf("update BuildRun status error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("first Reconcile() error = %v", err)
	}
	key := types.NamespacedName{Name: "sample-buildrun-sample-environment", Namespace: buildRun.Namespace}
	release := &cicdv1alpha1.Release{}
	if err := reconciler.Get(ctx, key, release); err != nil {
		t.Fatalf("get Release error = %v", err)
	}
	if err := reconciler.Delete(ctx, release); err != nil {
		t.Fatalf("delete Release error = %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() after Release deletion error = %v", err)
	}
	if err := reconciler.Get(ctx, key, &cicdv1alpha1.Release{}); err != nil {
		t.Fatalf("get recreated Release error = %v", err)
	}
}

func TestBuildRunRunningEventIsNotRepeated(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	recorder := &countingEventRecorder{reasons: map[string]int{}}
	reconciler.Recorder = recorder
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("initial Reconcile() error = %v", err)
	}
	job := getRunnerJob(t, ctx, reconciler, buildRun)
	job.Status.Active = 1
	if err := reconciler.Status().Update(ctx, job); err != nil {
		t.Fatalf("update Job status error = %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
			t.Fatalf("running Reconcile() iteration %d error = %v", i+1, err)
		}
	}
	if recorder.reasons["BuildStarted"] != 1 {
		t.Fatalf("BuildStarted events = %d, want 1", recorder.reasons["BuildStarted"])
	}
}

func TestBuildRunReconcileMarksFailed(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	job := getRunnerJob(t, ctx, reconciler, buildRun)
	job.Status.Failed = 1
	if err := reconciler.Status().Update(ctx, job); err != nil {
		t.Fatalf("update Job status error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() after failure error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed {
		t.Fatalf("phase = %q, want Failed", updated.Status.Phase)
	}
	if updated.Status.Failure.Reason != "JobFailed" {
		t.Fatalf("failure reason = %q, want JobFailed", updated.Status.Failure.Reason)
	}
}

func TestBuildRunReconcileGeneratesDeterministicImageTag(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Spec.Image.Tag = ""
	buildRun.Spec.CommitSHA = "abcdef0123456789"
	buildRun.Spec.Branch = "Feature/Payments"
	if err := reconciler.Update(ctx, buildRun); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatal(err)
	}
	if updated.Spec.Image.Tag != "feature-payments-abcdef012345" {
		t.Fatalf("image tag = %q", updated.Spec.Image.Tag)
	}
	if updated.Status.Image == nil || updated.Status.Image.Tag != updated.Spec.Image.Tag {
		t.Fatalf("status image = %#v, want generated tag %q", updated.Status.Image, updated.Spec.Image.Tag)
	}
	job := getRunnerJob(t, ctx, reconciler, updated)
	if got := envValue(job.Spec.Template.Spec.Containers[0].Env, "IMAGE_TAG"); got != updated.Spec.Image.Tag {
		t.Fatalf("Job IMAGE_TAG = %q, want %q", got, updated.Spec.Image.Tag)
	}
}

func TestBuildRunReconcileRejectsInvalidImageTagTemplate(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Spec.Image.Tag = ""
	if err := reconciler.Update(ctx, buildRun); err != nil {
		t.Fatal(err)
	}
	project := &cicdv1alpha1.Project{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: buildRun.Spec.ProjectRef, Namespace: buildRun.Namespace}, project); err != nil {
		t.Fatal(err)
	}
	project.Spec.ImageTagPolicy = &cicdv1alpha1.ProjectImageTagPolicySpec{DefaultTagTemplate: "{{ .Unknown }}"}
	if err := reconciler.Update(ctx, project); err != nil {
		t.Fatal(err)
	}
	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Failure.Reason != "ImageTagInvalid" || updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed {
		t.Fatalf("BuildRun status = %#v", updated.Status)
	}
}

func TestBuildRunReconcileFailsWhenProjectNamespaceDiffers(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	project := &cicdv1alpha1.Project{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: buildRun.Spec.ProjectRef, Namespace: buildRun.Namespace}, project); err != nil {
		t.Fatalf("get Project error = %v", err)
	}
	project.Spec.Namespace = "other-namespace"
	if err := reconciler.Update(ctx, project); err != nil {
		t.Fatalf("update Project error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed {
		t.Fatalf("phase = %q, want Failed", updated.Status.Phase)
	}
	if !strings.Contains(updated.Status.Failure.Message, "must match BuildRun namespace") {
		t.Fatalf("failure message = %q", updated.Status.Failure.Message)
	}
}

func TestBuildRunReconcileReportsMissingRegistryCredentials(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	project := &cicdv1alpha1.Project{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: buildRun.Spec.ProjectRef, Namespace: buildRun.Namespace}, project); err != nil {
		t.Fatal(err)
	}
	project.Spec.Registry = &cicdv1alpha1.ProjectRegistrySpec{
		Provider:            cicdv1alpha1.RegistryProviderGHCR,
		CredentialSecretRef: &cicdv1alpha1.SecretKeyRef{Name: "missing-registry-auth"},
	}
	if err := reconciler.Update(ctx, project); err != nil {
		t.Fatal(err)
	}
	template := &cicdv1alpha1.PipelineTemplate{}
	if err := reconciler.Get(ctx, types.NamespacedName{Name: buildRun.Spec.PipelineTemplateRef, Namespace: buildRun.Namespace}, template); err != nil {
		t.Fatal(err)
	}
	template.Spec.Build.Enabled = true
	template.Spec.Build.Builder = cicdv1alpha1.BuildBuilderBuildKit
	template.Spec.Build.Push = true
	if err := reconciler.Update(ctx, template); err != nil {
		t.Fatal(err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatal(err)
	}
	if updated.Status.Failure.Reason != "RegistryCredentialsMissing" || !strings.Contains(updated.Status.Failure.Message, "missing-registry-auth") {
		t.Fatalf("failure = %#v", updated.Status.Failure)
	}
}

func TestTerminalBuildRunDoesNotCreateJob(t *testing.T) {
	ctx := context.Background()
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Status.Phase = cicdv1alpha1.BuildRunPhaseSucceeded
	if err := reconciler.Status().Update(ctx, buildRun); err != nil {
		t.Fatalf("update BuildRun status error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	jobs := &batchv1.JobList{}
	if err := reconciler.List(ctx, jobs, client.InNamespace(buildRun.Namespace)); err != nil {
		t.Fatalf("List Jobs error = %v", err)
	}
	if len(jobs.Items) != 0 {
		t.Fatalf("len(jobs.Items) = %d, want 0", len(jobs.Items))
	}
}

func TestTektonExecutorRequiresFeatureFlag(t *testing.T) {
	ctx := context.Background()
	t.Setenv("CLOU_DIVISION_ENABLE_TEKTON", "false")
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Spec.Executor = cicdv1alpha1.ExecutorTypeTekton
	if err := reconciler.Update(ctx, buildRun); err != nil {
		t.Fatalf("update BuildRun error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseFailed {
		t.Fatalf("phase = %q, want Failed", updated.Status.Phase)
	}
	if updated.Status.Failure.Reason != "TektonUnavailable" {
		t.Fatalf("failure reason = %q, want TektonUnavailable", updated.Status.Failure.Reason)
	}
}

func TestTektonExecutorSelectionCreatesPipelineRun(t *testing.T) {
	ctx := context.Background()
	t.Setenv("CLOU_DIVISION_ENABLE_TEKTON", "true")
	reconciler, buildRun := newBuildRunReconciler(t)
	buildRun.Spec.Executor = cicdv1alpha1.ExecutorTypeTekton
	if err := reconciler.Update(ctx, buildRun); err != nil {
		t.Fatalf("update BuildRun error = %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, requestFor(buildRun)); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &cicdv1alpha1.BuildRun{}
	if err := reconciler.Get(ctx, client.ObjectKeyFromObject(buildRun), updated); err != nil {
		t.Fatalf("get BuildRun error = %v", err)
	}
	if updated.Status.Phase != cicdv1alpha1.BuildRunPhaseQueued {
		t.Fatalf("phase = %q, want Queued", updated.Status.Phase)
	}
	if updated.Status.PipelineRunRef.Name == "" {
		t.Fatalf("pipelineRunRef is empty")
	}
	jobs := &batchv1.JobList{}
	if err := reconciler.List(ctx, jobs, client.InNamespace(buildRun.Namespace)); err != nil {
		t.Fatalf("List Jobs error = %v", err)
	}
	if len(jobs.Items) != 0 {
		t.Fatalf("len(jobs.Items) = %d, want 0", len(jobs.Items))
	}
}

func newBuildRunReconciler(t *testing.T) (*BuildRunReconciler, *cicdv1alpha1.BuildRun) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add cloudivision scheme: %v", err)
	}

	project := testProject()
	repository := testRepository()
	template := testPipelineTemplate()
	buildRun := testBuildRun()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}, &cicdv1alpha1.Release{}, &batchv1.Job{}).
		WithObjects(project, repository, template, buildRun).
		Build()

	return &BuildRunReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}, buildRun
}

func requestFor(buildRun *cicdv1alpha1.BuildRun) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      buildRun.Name,
			Namespace: buildRun.Namespace,
		},
	}
}

func testProject() *cicdv1alpha1.Project {
	return &cicdv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-project", Namespace: "ci"},
		Spec: cicdv1alpha1.ProjectSpec{
			DisplayName:        "Sample Project",
			OwnerTeam:          "platform",
			Namespace:          "ci",
			DefaultRegistry:    "ghcr.io/cloudivision",
			DefaultBranch:      "main",
			ServiceAccountName: "sample-builder",
			Isolation: cicdv1alpha1.ProjectIsolation{
				CreateNamespace:   true,
				PodSecurityLevel:  cicdv1alpha1.PodSecurityLevelRestricted,
				NetworkPolicyMode: cicdv1alpha1.NetworkPolicyModeDefaultDeny,
			},
		},
	}
}

func testRepository() *cicdv1alpha1.Repository {
	return &cicdv1alpha1.Repository{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-repository", Namespace: "ci"},
		Spec: cicdv1alpha1.RepositorySpec{
			ProjectRef:          "sample-project",
			Provider:            cicdv1alpha1.RepositoryProviderGitHub,
			URL:                 "https://github.com/cloudivision/example.git",
			DefaultBranch:       "main",
			PipelineTemplateRef: "sample-template",
		},
	}
}

func testPipelineTemplate() *cicdv1alpha1.PipelineTemplate {
	return &cicdv1alpha1.PipelineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-template", Namespace: "ci"},
		Spec: cicdv1alpha1.PipelineTemplateSpec{
			Resources: cicdv1alpha1.PipelineResourceSpec{
				CPURequest:     "100m",
				CPULimit:       "500m",
				MemoryRequest:  "128Mi",
				MemoryLimit:    "512Mi",
				TimeoutSeconds: 600,
			},
			Security: cicdv1alpha1.PipelineSecuritySpec{
				AllowPrivileged:        false,
				RunAsNonRoot:           true,
				ReadOnlyRootFilesystem: false,
			},
		},
	}
}

func testBuildRun() *cicdv1alpha1.BuildRun {
	return &cicdv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{Name: "sample-buildrun", Namespace: "ci"},
		Spec: cicdv1alpha1.BuildRunSpec{
			ProjectRef:          "sample-project",
			RepositoryRef:       "sample-repository",
			PipelineTemplateRef: "sample-template",
			Revision:            "main",
			Branch:              "main",
			TriggeredBy: cicdv1alpha1.TriggeredBy{
				Type:  cicdv1alpha1.TriggerTypeManual,
				Actor: "developer",
			},
			Image: cicdv1alpha1.ImageRef{
				Repository: "ghcr.io/cloudivision/example",
				Tag:        "main",
			},
			Executor: cicdv1alpha1.ExecutorTypeJob,
			GitOps: cicdv1alpha1.BuildRunGitOpsSpec{
				Enabled:        true,
				Strategy:       cicdv1alpha1.GitOpsStrategyKustomizeImage,
				EnvironmentRef: "sample-environment",
			},
		},
	}
}

func getRunnerJob(t *testing.T, ctx context.Context, reader client.Reader, buildRun *cicdv1alpha1.BuildRun) *batchv1.Job {
	t.Helper()
	job := &batchv1.Job{}
	key := types.NamespacedName{Name: jobexecutor.NameForBuildRun(buildRun.Name), Namespace: buildRun.Namespace}
	if err := reader.Get(ctx, key, job); err != nil {
		t.Fatalf("get Job error = %v", err)
	}
	return job
}

func envValue(env []corev1.EnvVar, name string) string {
	for _, item := range env {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}

func assertSecureJobSpec(t *testing.T, job *batchv1.Job) {
	t.Helper()
	podSpec := job.Spec.Template.Spec
	if len(podSpec.Volumes) != 0 {
		t.Fatalf("volumes = %#v, want none", podSpec.Volumes)
	}
	if podSpec.SecurityContext == nil || podSpec.SecurityContext.RunAsNonRoot == nil || !*podSpec.SecurityContext.RunAsNonRoot {
		t.Fatalf("pod runAsNonRoot is not true")
	}
	container := podSpec.Containers[0]
	if container.SecurityContext == nil {
		t.Fatal("container securityContext = nil")
	}
	if container.SecurityContext.Privileged == nil || *container.SecurityContext.Privileged {
		t.Fatal("container privileged is not explicitly false")
	}
	if container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Fatal("allowPrivilegeEscalation is not explicitly false")
	}
	if len(container.SecurityContext.Capabilities.Drop) != 1 || container.SecurityContext.Capabilities.Drop[0] != "ALL" {
		t.Fatalf("capabilities drop = %#v, want ALL", container.SecurityContext.Capabilities.Drop)
	}
}

type countingEventRecorder struct {
	reasons map[string]int
}

func (r *countingEventRecorder) Event(_ runtime.Object, _, reason, _ string) {
	r.reasons[reason]++
}

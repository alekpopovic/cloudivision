package controller

import (
	"context"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
	if updated.Status.Image.Repository != buildRun.Spec.Image.Repository {
		t.Fatalf("status image repository = %q", updated.Status.Image.Repository)
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
		WithStatusSubresource(&cicdv1alpha1.BuildRun{}, &batchv1.Job{}).
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
			Namespace:          "sample-project",
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
	key := types.NamespacedName{Name: jobNameForBuildRun(buildRun.Name), Namespace: buildRun.Namespace}
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

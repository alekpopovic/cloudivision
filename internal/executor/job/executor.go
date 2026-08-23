package job

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/executor"
	providerregistry "github.com/cloudivision/cloudivision/internal/provider/registry"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultRunnerImage                   = "cloudivision/runner:dev"
	defaultTTLSecondsAfterFinished int32 = 3600
	defaultBackoffLimit            int32 = 0
	defaultActiveDeadlineSeconds   int64 = 3600
	defaultRunnerServiceAccount          = "cloudivision-runner"
	defaultCPURequest                    = "100m"
	defaultCPULimit                      = "1000m"
	defaultMemoryRequest                 = "128Mi"
	defaultMemoryLimit                   = "1Gi"
	registryCredentialsVolume            = "registry-credentials"
	RegistryCredentialsDir               = "/var/run/secrets/cloudivision-registry"
)

var (
	ErrRegistryCredentialsMissing = errors.New("registry credentials are missing")
	ErrRegistryCredentialsInvalid = errors.New("registry credentials are invalid")
)

type Executor struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func (e Executor) EnsureRun(ctx context.Context, req executor.EnsureRunRequest) (*executor.RunRef, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	if req.Template.Spec.Security.AllowPrivileged && !allowPrivilegedBuilds() {
		return nil, fmt.Errorf("PipelineTemplate %q requests privileged build execution; set CLOU_DIVISION_ALLOW_PRIVILEGED_BUILDS=true to allow this unsupported mode", req.Template.Name)
	}
	if err := e.validateRegistryCredentials(ctx, req); err != nil {
		return nil, err
	}
	key := types.NamespacedName{
		Name:      NameForBuildRun(req.BuildRun.Name),
		Namespace: req.BuildRun.Namespace,
	}
	existing := &batchv1.Job{}
	if err := e.Client.Get(ctx, key, existing); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("get Job %s: %w", key, err)
		}
		job := buildJob(req.BuildRun, req.Project, req.Repository, req.Template)
		if err := controllerutil.SetControllerReference(req.BuildRun, job, e.Scheme); err != nil {
			return nil, fmt.Errorf("set Job owner reference: %w", err)
		}
		if err := e.Client.Create(ctx, job); err != nil && !apierrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("create Job %s: %w", key, err)
		}
	}
	return &executor.RunRef{Kind: "Job", Name: key.Name, Namespace: key.Namespace}, nil
}

func (e Executor) ReadRunStatus(ctx context.Context, ref executor.RunRef) (*executor.RunStatus, error) {
	job := &batchv1.Job{}
	key := types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
	if err := e.Client.Get(ctx, key, job); err != nil {
		return nil, fmt.Errorf("get Job %s: %w", key, err)
	}
	if job.Status.Succeeded > 0 {
		return &executor.RunStatus{Phase: executor.RunPhaseSucceeded}, nil
	}
	if jobFailed(job) {
		return &executor.RunStatus{
			Phase: executor.RunPhaseFailed,
			Failure: executor.Failure{
				Reason:  "JobFailed",
				Message: "Runner Job failed",
			},
		}, nil
	}
	if job.Status.Active > 0 {
		return &executor.RunStatus{Phase: executor.RunPhaseRunning}, nil
	}
	return &executor.RunStatus{Phase: executor.RunPhasePending}, nil
}

func (e Executor) CancelRun(ctx context.Context, ref executor.RunRef) error {
	job := &batchv1.Job{}
	key := types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
	if err := e.Client.Get(ctx, key, job); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get Job %s for cancellation: %w", key, err)
	}
	if err := e.Client.Delete(ctx, job); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete Job %s: %w", key, err)
	}
	return nil
}

func buildJob(buildRun *cicdv1alpha1.BuildRun, project *cicdv1alpha1.Project, repository *cicdv1alpha1.Repository, template *cicdv1alpha1.PipelineTemplate) *batchv1.Job {
	labels := map[string]string{
		"app.kubernetes.io/name":      "cloudivision",
		"app.kubernetes.io/component": "runner",
		"cloudivision.io/buildrun":    buildRun.Name,
		"cloudivision.io/project":     project.Name,
	}
	backoffLimit := defaultBackoffLimit
	ttlSeconds := defaultTTLSecondsAfterFinished
	var activeDeadlineSeconds *int64
	if template.Spec.Resources.TimeoutSeconds > 0 {
		timeout := int64(template.Spec.Resources.TimeoutSeconds)
		activeDeadlineSeconds = &timeout
	}
	if activeDeadlineSeconds == nil {
		timeout := defaultActiveDeadlineSeconds
		activeDeadlineSeconds = &timeout
	}
	runAsNonRoot := true
	automountServiceAccountToken := true
	allowPrivilegeEscalation := false
	privileged := false
	seccompProfile := corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	if ref := template.Spec.SupplyChain.SigningKeySecretRef; ref != nil {
		mode := int32(0o400)
		volumes = append(volumes, corev1.Volume{
			Name: "cosign-key",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
				SecretName: ref.Name,
				Items:      []corev1.KeyToPath{{Key: ref.Key, Path: "key", Mode: &mode}},
			}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "cosign-key", MountPath: "/var/run/secrets/cloudivision-signing", ReadOnly: true})
	}
	if registryCredentialRequired(project, template) && project.Spec.Registry.CredentialSecretRef != nil {
		ref := project.Spec.Registry.CredentialSecretRef
		mode := int32(0o400)
		secretVolume := &corev1.SecretVolumeSource{SecretName: ref.Name, DefaultMode: &mode}
		if ref.Key != "" {
			secretVolume.Items = []corev1.KeyToPath{{Key: ref.Key, Path: providerregistry.DockerConfigJSONKey, Mode: &mode}}
		}
		volumes = append(volumes, corev1.Volume{
			Name:         registryCredentialsVolume,
			VolumeSource: corev1.VolumeSource{Secret: secretVolume},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: registryCredentialsVolume, MountPath: RegistryCredentialsDir, ReadOnly: true})
	}
	if os.Getenv("CLOU_DIVISION_LOG_BACKEND") == "local" && os.Getenv("CLOU_DIVISION_LOG_PVC") != "" {
		volumes = append(volumes, corev1.Volume{Name: "stored-logs", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: os.Getenv("CLOU_DIVISION_LOG_PVC")}}})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "stored-logs", MountPath: defaultString(os.Getenv("CLOU_DIVISION_LOG_ROOT"), "/var/lib/cloudivision/logs")})
	}
	if os.Getenv("CLOU_DIVISION_ARTIFACT_BACKEND") == "local" && os.Getenv("CLOU_DIVISION_ARTIFACT_PVC") != "" {
		volumes = append(volumes, corev1.Volume{Name: "stored-artifacts", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: os.Getenv("CLOU_DIVISION_ARTIFACT_PVC")}}})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "stored-artifacts", MountPath: defaultString(os.Getenv("CLOU_DIVISION_ARTIFACT_ROOT"), "/var/lib/cloudivision/artifacts")})
	}
	if template.Spec.Cache.Enabled && template.Spec.Cache.Mode == cicdv1alpha1.DependencyCacheModePVC && os.Getenv("CLOU_DIVISION_CACHE_PVC") != "" {
		volumes = append(volumes, corev1.Volume{Name: "dependency-cache", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: os.Getenv("CLOU_DIVISION_CACHE_PVC")}}})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: "dependency-cache", MountPath: defaultString(os.Getenv("CLOU_DIVISION_CACHE_ROOT"), "/var/lib/cloudivision/cache")})
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NameForBuildRun(buildRun.Name),
			Namespace: buildRun.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy:                corev1.RestartPolicyNever,
					ServiceAccountName:           serviceAccountName(project),
					AutomountServiceAccountToken: &automountServiceAccountToken,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot:   &runAsNonRoot,
						SeccompProfile: &seccompProfile,
					},
					Volumes: volumes,
					Containers: []corev1.Container{
						{
							Name:         "runner",
							Image:        runnerImage(),
							Env:          runnerEnv(buildRun, project, repository, template),
							Resources:    resourceRequirements(template.Spec.Resources),
							VolumeMounts: volumeMounts,
							SecurityContext: &corev1.SecurityContext{
								RunAsNonRoot:             &runAsNonRoot,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								Privileged:               &privileged,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func NameForBuildRun(buildRunName string) string {
	name := strings.ToLower(buildRunName) + "-runner"
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

func runnerImage() string {
	if image := os.Getenv("CLOU_DIVISION_RUNNER_IMAGE"); image != "" {
		return image
	}
	return DefaultRunnerImage
}

func runnerEnv(buildRun *cicdv1alpha1.BuildRun, project *cicdv1alpha1.Project, repository *cicdv1alpha1.Repository, template *cicdv1alpha1.PipelineTemplate) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "BUILD_RUN_NAME", Value: buildRun.Name},
		{Name: "BUILD_RUN_NAMESPACE", Value: buildRun.Namespace},
		{Name: "PROJECT_NAME", Value: project.Name},
		{Name: "REPOSITORY_URL", Value: repository.Spec.URL},
		{Name: "REVISION", Value: buildRun.Spec.Revision},
		{Name: "BRANCH", Value: buildRun.Spec.Branch},
		{Name: "PIPELINE_TEMPLATE_NAME", Value: buildRun.Spec.PipelineTemplateRef},
		{Name: "IMAGE_REPOSITORY", Value: buildRun.Spec.Image.Repository},
		{Name: "IMAGE_TAG", Value: buildRun.Spec.Image.Tag},
		{Name: "GITOPS_ENABLED", Value: strconv.FormatBool(buildRun.Spec.GitOps.Enabled)},
	}
	if project.Spec.Registry != nil && project.Spec.Registry.Provider != "" {
		imagePrefix := project.Spec.Registry.ImagePrefix
		if imagePrefix == "" {
			imagePrefix = project.Spec.DefaultRegistry
		}
		env = append(env,
			corev1.EnvVar{Name: "REGISTRY_PROVIDER", Value: string(project.Spec.Registry.Provider)},
			corev1.EnvVar{Name: "REGISTRY_IMAGE_PREFIX", Value: imagePrefix},
		)
		if project.Spec.Registry.CredentialSecretRef != nil && project.Spec.Registry.CredentialSecretRef.Name != "" {
			env = append(env, corev1.EnvVar{Name: "REGISTRY_CREDENTIALS_DIR", Value: RegistryCredentialsDir})
		}
	}
	if project.Spec.Quotas != nil {
		if project.Spec.Quotas.MaxArtifactsSize != "" {
			env = append(env, corev1.EnvVar{Name: "PROJECT_MAX_ARTIFACTS_SIZE", Value: project.Spec.Quotas.MaxArtifactsSize})
		}
		if project.Spec.Quotas.MaxLogSize != "" {
			env = append(env, corev1.EnvVar{Name: "PROJECT_MAX_LOG_SIZE", Value: project.Spec.Quotas.MaxLogSize})
		}
	}
	if backend := os.Getenv("CLOU_DIVISION_LOG_BACKEND"); backend != "" {
		env = append(env, corev1.EnvVar{Name: "LOG_BACKEND", Value: backend})
	}
	if root := os.Getenv("CLOU_DIVISION_LOG_ROOT"); root != "" {
		env = append(env, corev1.EnvVar{Name: "LOG_ROOT", Value: root})
	}
	if backend := os.Getenv("CLOU_DIVISION_ARTIFACT_BACKEND"); backend != "" {
		env = append(env, corev1.EnvVar{Name: "ARTIFACT_BACKEND", Value: backend})
	}
	if root := os.Getenv("CLOU_DIVISION_ARTIFACT_ROOT"); root != "" {
		env = append(env, corev1.EnvVar{Name: "ARTIFACT_ROOT", Value: root})
	}
	if template.Spec.Cache.Enabled {
		switch template.Spec.Cache.Mode {
		case cicdv1alpha1.DependencyCacheModePVC:
			if os.Getenv("CLOU_DIVISION_CACHE_PVC") != "" {
				env = append(env, corev1.EnvVar{Name: "CACHE_BACKEND", Value: "pvc"})
			}
		case cicdv1alpha1.DependencyCacheModeObjectStorage:
			env = append(env, corev1.EnvVar{Name: "CACHE_BACKEND", Value: "object-storage"})
		}
	}
	if root := os.Getenv("CLOU_DIVISION_CACHE_ROOT"); root != "" {
		env = append(env, corev1.EnvVar{Name: "CACHE_ROOT", Value: root})
	}
	if maximum := os.Getenv("CLOU_DIVISION_CACHE_MAX_SIZE"); maximum != "" {
		env = append(env, corev1.EnvVar{Name: "CACHE_MAX_SIZE", Value: maximum})
	}
	return env
}

func resourceRequirements(resources cicdv1alpha1.PipelineResourceSpec) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	addQuantity(requirements.Requests, corev1.ResourceCPU, defaultString(resources.CPURequest, defaultCPURequest))
	addQuantity(requirements.Limits, corev1.ResourceCPU, defaultString(resources.CPULimit, defaultCPULimit))
	addQuantity(requirements.Requests, corev1.ResourceMemory, defaultString(resources.MemoryRequest, defaultMemoryRequest))
	addQuantity(requirements.Limits, corev1.ResourceMemory, defaultString(resources.MemoryLimit, defaultMemoryLimit))
	return requirements
}

func addQuantity(list corev1.ResourceList, name corev1.ResourceName, value string) {
	if value == "" {
		return
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return
	}
	list[name] = quantity
}

func jobFailed(job *batchv1.Job) bool {
	if job.Status.Failed == 0 {
		return false
	}
	if job.Spec.BackoffLimit == nil {
		return job.Status.Failed > defaultBackoffLimit
	}
	return job.Status.Failed > *job.Spec.BackoffLimit
}

func validateRequest(req executor.EnsureRunRequest) error {
	if req.BuildRun == nil || req.Project == nil || req.Repository == nil || req.Template == nil {
		return fmt.Errorf("job executor request requires BuildRun, Project, Repository and PipelineTemplate")
	}
	if req.BuildRun.Name == "" || req.BuildRun.Namespace == "" {
		return fmt.Errorf("BuildRun name and namespace are required")
	}
	return nil
}

func (e Executor) validateRegistryCredentials(ctx context.Context, req executor.EnsureRunRequest) error {
	if !registryCredentialRequired(req.Project, req.Template) {
		return nil
	}
	ref := req.Project.Spec.Registry.CredentialSecretRef
	if ref == nil || ref.Name == "" {
		return fmt.Errorf("Project %q registry credentialSecretRef is required for image push: %w", req.Project.Name, ErrRegistryCredentialsMissing)
	}
	secret := &corev1.Secret{}
	key := types.NamespacedName{Name: ref.Name, Namespace: req.BuildRun.Namespace}
	if err := e.Client.Get(ctx, key, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("registry credential Secret %q was not found in namespace %q: %w", ref.Name, req.BuildRun.Namespace, ErrRegistryCredentialsMissing)
		}
		return fmt.Errorf("get registry credential Secret %s: %w", key, err)
	}
	data := secret.Data
	if ref.Key != "" {
		value, ok := data[ref.Key]
		if !ok {
			return fmt.Errorf("registry credential Secret %q does not contain key %q: %w", ref.Name, ref.Key, ErrRegistryCredentialsInvalid)
		}
		data = map[string][]byte{providerregistry.DockerConfigJSONKey: value}
	}
	if _, err := providerregistry.ParseCredential(data); err != nil {
		return fmt.Errorf("registry credential Secret %q has no supported credential data: %w: %w", ref.Name, ErrRegistryCredentialsInvalid, err)
	}
	return nil
}

func registryCredentialRequired(project *cicdv1alpha1.Project, template *cicdv1alpha1.PipelineTemplate) bool {
	return project != nil && template != nil && project.Spec.Registry != nil && project.Spec.Registry.Provider != "" && template.Spec.Build.Enabled && template.Spec.Build.Push
}

func serviceAccountName(project *cicdv1alpha1.Project) string {
	if project.Spec.ServiceAccountName != "" {
		return project.Spec.ServiceAccountName
	}
	return defaultRunnerServiceAccount
}

func allowPrivilegedBuilds() bool {
	return strings.EqualFold(os.Getenv("CLOU_DIVISION_ALLOW_PRIVILEGED_BUILDS"), "true")
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

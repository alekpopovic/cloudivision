package tekton

import (
	"context"
	"errors"
	"fmt"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/executor"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var ErrTektonUnavailable = errors.New("tekton PipelineRun CRD is not available")

type Executor struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func (e Executor) EnsureRun(ctx context.Context, req executor.EnsureRunRequest) (*executor.RunRef, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	name := NameForBuildRun(req.BuildRun.Name)
	key := types.NamespacedName{Name: name, Namespace: req.BuildRun.Namespace}
	existing := pipelineRun()
	if err := e.Client.Get(ctx, key, existing); err != nil {
		if !apierrors.IsNotFound(err) {
			if meta.IsNoMatchError(err) {
				return nil, fmt.Errorf("%w: %v", ErrTektonUnavailable, err)
			}
			return nil, fmt.Errorf("get Tekton PipelineRun %s: %w", key, err)
		}
		run := buildPipelineRun(req.BuildRun, req.Project, req.Repository, req.Template)
		if err := controllerutil.SetControllerReference(req.BuildRun, run, e.Scheme); err != nil {
			return nil, fmt.Errorf("set PipelineRun owner reference: %w", err)
		}
		if err := e.Client.Create(ctx, run); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return &executor.RunRef{Kind: "PipelineRun", Name: name, Namespace: req.BuildRun.Namespace}, nil
			}
			if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("%w: %v", ErrTektonUnavailable, err)
			}
			return nil, fmt.Errorf("create Tekton PipelineRun %s: %w", key, err)
		}
	}
	return &executor.RunRef{Kind: "PipelineRun", Name: name, Namespace: req.BuildRun.Namespace}, nil
}

func (e Executor) ReadRunStatus(ctx context.Context, ref executor.RunRef) (*executor.RunStatus, error) {
	run := pipelineRun()
	key := types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
	if err := e.Client.Get(ctx, key, run); err != nil {
		if meta.IsNoMatchError(err) {
			return nil, fmt.Errorf("%w: %v", ErrTektonUnavailable, err)
		}
		return nil, fmt.Errorf("get Tekton PipelineRun %s: %w", key, err)
	}
	conditions, _, _ := unstructured.NestedSlice(run.Object, "status", "conditions")
	for _, raw := range conditions {
		condition, ok := raw.(map[string]any)
		if !ok || condition["type"] != "Succeeded" {
			continue
		}
		reason, _ := condition["reason"].(string)
		message, _ := condition["message"].(string)
		switch condition["status"] {
		case "True":
			return &executor.RunStatus{Phase: executor.RunPhaseSucceeded}, nil
		case "False":
			return &executor.RunStatus{
				Phase: executor.RunPhaseFailed,
				Failure: executor.Failure{
					Reason:  reasonOrDefault(reason, "PipelineRunFailed"),
					Message: messageOrDefault(message, "Tekton PipelineRun failed"),
				},
			}, nil
		}
	}
	return &executor.RunStatus{Phase: executor.RunPhaseRunning}, nil
}

func (e Executor) CancelRun(ctx context.Context, ref executor.RunRef) error {
	run := pipelineRun()
	key := types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}
	if err := e.Client.Get(ctx, key, run); err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}
		return fmt.Errorf("get Tekton PipelineRun %s for cancellation: %w", key, err)
	}
	annotations := run.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["tekton.dev/cancelled"] = "true"
	run.SetAnnotations(annotations)
	spec, _, _ := unstructured.NestedMap(run.Object, "spec")
	spec["status"] = "PipelineRunCancelled"
	if err := unstructured.SetNestedMap(run.Object, spec, "spec"); err != nil {
		return fmt.Errorf("set PipelineRun cancellation status: %w", err)
	}
	if err := e.Client.Update(ctx, run); err != nil {
		return fmt.Errorf("cancel Tekton PipelineRun %s: %w", key, err)
	}
	return nil
}

func buildPipelineRun(buildRun *cicdv1alpha1.BuildRun, project *cicdv1alpha1.Project, repository *cicdv1alpha1.Repository, template *cicdv1alpha1.PipelineTemplate) *unstructured.Unstructured {
	run := pipelineRun()
	run.SetName(NameForBuildRun(buildRun.Name))
	run.SetNamespace(buildRun.Namespace)
	run.SetLabels(map[string]string{
		"app.kubernetes.io/name":      "cloudivision",
		"app.kubernetes.io/component": "tekton-runner",
		"cloudivision.io/buildrun":    buildRun.Name,
		"cloudivision.io/project":     project.Name,
	})
	run.Object["spec"] = map[string]any{
		"params": []any{
			map[string]any{"name": "repository-url", "value": repository.Spec.URL},
			map[string]any{"name": "revision", "value": buildRun.Spec.Revision},
			map[string]any{"name": "branch", "value": buildRun.Spec.Branch},
			map[string]any{"name": "image-repository", "value": buildRun.Spec.Image.Repository},
			map[string]any{"name": "image-tag", "value": buildRun.Spec.Image.Tag},
		},
		"pipelineSpec": map[string]any{
			"params": []any{
				map[string]any{"name": "repository-url", "type": "string"},
				map[string]any{"name": "revision", "type": "string"},
				map[string]any{"name": "branch", "type": "string"},
				map[string]any{"name": "image-repository", "type": "string"},
				map[string]any{"name": "image-tag", "type": "string"},
			},
			"tasks": tektonTasks(template),
		},
	}
	return run
}

func tektonTasks(template *cicdv1alpha1.PipelineTemplate) []any {
	tasks := make([]any, 0, len(template.Spec.Steps))
	previous := ""
	for i, step := range template.Spec.Steps {
		name := sanitizeName(step.Name)
		if name == "" {
			name = fmt.Sprintf("step-%d", i+1)
		}
		task := map[string]any{
			"name": name,
			"taskSpec": map[string]any{
				"steps": []any{tektonStep(step)},
			},
		}
		if previous != "" {
			task["runAfter"] = []any{previous}
		}
		tasks = append(tasks, task)
		previous = name
	}
	if template.Spec.Build.Enabled {
		task := map[string]any{
			"name": "build-image",
			"taskSpec": map[string]any{
				"params": []any{
					map[string]any{"name": "image-repository", "type": "string"},
					map[string]any{"name": "image-tag", "type": "string"},
				},
				"steps": []any{
					map[string]any{
						"name":  "build",
						"image": builderImage(template.Spec.Build.Builder),
						"env": []any{
							map[string]any{"name": "IMAGE_REPOSITORY", "value": "$(params.image-repository)"},
							map[string]any{"name": "IMAGE_TAG", "value": "$(params.image-tag)"},
							map[string]any{"name": "CONTEXT_DIR", "value": defaultString(template.Spec.Build.ContextDir, ".")},
							map[string]any{"name": "DOCKERFILE", "value": defaultString(template.Spec.Build.Dockerfile, "Dockerfile")},
						},
						"script": "#!/bin/sh\nset -eu\necho \"Tekton image build adapter placeholder for ${IMAGE_REPOSITORY}:${IMAGE_TAG}\"\n",
					},
				},
			},
			"params": []any{
				map[string]any{"name": "image-repository", "value": "$(params.image-repository)"},
				map[string]any{"name": "image-tag", "value": "$(params.image-tag)"},
			},
		}
		if previous != "" {
			task["runAfter"] = []any{previous}
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func tektonStep(step cicdv1alpha1.PipelineStep) map[string]any {
	result := map[string]any{
		"name":  sanitizeName(step.Name),
		"image": step.Image,
	}
	if len(step.Command) > 0 {
		result["command"] = stringSliceToAny(step.Command)
	}
	if len(step.Args) > 0 {
		result["args"] = stringSliceToAny(step.Args)
	}
	if step.WorkingDir != "" {
		result["workingDir"] = step.WorkingDir
	}
	if len(step.Env) > 0 {
		env := make([]any, 0, len(step.Env))
		for _, item := range step.Env {
			env = append(env, map[string]any{"name": item.Name, "value": item.Value})
		}
		result["env"] = env
	}
	return result
}

func pipelineRun() *unstructured.Unstructured {
	run := &unstructured.Unstructured{}
	run.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "tekton.dev",
		Version: "v1",
		Kind:    "PipelineRun",
	})
	return run
}

func NameForBuildRun(buildRunName string) string {
	name := strings.ToLower(buildRunName) + "-pipelinerun"
	name = strings.NewReplacer("_", "-", ".", "-").Replace(name)
	if len(name) <= 63 {
		return name
	}
	return name[:63]
}

func validateRequest(req executor.EnsureRunRequest) error {
	if req.BuildRun == nil || req.Project == nil || req.Repository == nil || req.Template == nil {
		return fmt.Errorf("tekton executor request requires BuildRun, Project, Repository and PipelineTemplate")
	}
	if req.BuildRun.Name == "" || req.BuildRun.Namespace == "" {
		return fmt.Errorf("BuildRun name and namespace are required")
	}
	return nil
}

func sanitizeName(value string) string {
	value = strings.ToLower(value)
	value = strings.NewReplacer("_", "-", ".", "-").Replace(value)
	value = strings.Trim(value, "-")
	if len(value) <= 63 {
		return value
	}
	return value[:63]
}

func stringSliceToAny(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func builderImage(builder cicdv1alpha1.BuildBuilder) string {
	switch builder {
	case cicdv1alpha1.BuildBuilderBuildah:
		return "quay.io/buildah/stable:latest"
	case cicdv1alpha1.BuildBuilderBuildKit:
		return "moby/buildkit:rootless"
	default:
		return "busybox:1.36"
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func reasonOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func messageOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

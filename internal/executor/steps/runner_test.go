package steps

import (
	"context"
	"strings"
	"testing"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/runner/redact"
	corev1 "k8s.io/api/core/v1"
)

func TestRunnerRedactsCommandFailureOutput(t *testing.T) {
	stepRunner := Runner{}
	err := stepRunner.Run(context.Background(), t.TempDir(), []cicdv1alpha1.PipelineStep{
		{
			Name:    "fail",
			Command: []string{"sh"},
			Args:    []string{"-c", "echo $SECRET_TOKEN && exit 7"},
			Env: []corev1.EnvVar{
				{Name: "SECRET_TOKEN", Value: "top-secret"},
			},
		},
	}, redact.FromEnv(map[string]string{"SECRET_TOKEN": "top-secret"}))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if strings.Contains(err.Error(), "top-secret") {
		t.Fatalf("error leaked secret value: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("error = %q, want redacted marker", err.Error())
	}
}

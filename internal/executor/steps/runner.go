package steps

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/redact"
)

type Runner struct {
	Output io.Writer
}

func (r Runner) Run(ctx context.Context, sourceDir string, pipelineSteps []cicdv1alpha1.PipelineStep, redactor redact.Redactor) error {
	for _, step := range pipelineSteps {
		if err := r.runStep(ctx, sourceDir, step, redactor); err != nil {
			if step.ContinueOnError {
				continue
			}
			return err
		}
	}
	return nil
}

func (r Runner) runStep(ctx context.Context, sourceDir string, step cicdv1alpha1.PipelineStep, redactor redact.Redactor) error {
	if step.Name == "" {
		return fmt.Errorf("pipeline step name is required")
	}
	if len(step.Command) == 0 {
		return fmt.Errorf("step %q command is required", step.Name)
	}
	stepCtx := ctx
	cancel := func() {}
	if step.TimeoutSeconds > 0 {
		stepCtx, cancel = context.WithTimeout(ctx, time.Duration(step.TimeoutSeconds)*time.Second)
	}
	defer cancel()

	cmd := exec.CommandContext(stepCtx, step.Command[0], append(step.Command[1:], step.Args...)...)
	cmd.Dir = workingDir(sourceDir, step.WorkingDir)
	cmd.Env = os.Environ()
	for _, env := range step.Env {
		if env.Value != "" {
			cmd.Env = append(cmd.Env, env.Name+"="+env.Value)
		}
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	redacted := redactor.Mask(output.String())
	if r.Output != nil && redacted != "" {
		_, _ = r.Output.Write([]byte(redacted))
	}
	if err != nil {
		return fmt.Errorf("step %q failed: %s: %w", step.Name, redacted, err)
	}
	return nil
}

func workingDir(sourceDir, stepWorkingDir string) string {
	if stepWorkingDir == "" {
		return sourceDir
	}
	if filepath.IsAbs(stepWorkingDir) {
		return stepWorkingDir
	}
	return filepath.Join(sourceDir, stepWorkingDir)
}

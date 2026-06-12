package executor

import (
	"context"
	"testing"
)

type fakePipelineExecutor struct{}

func (fakePipelineExecutor) EnsureRun(context.Context, EnsureRunRequest) (*RunRef, error) {
	return &RunRef{Kind: "Job", Name: "run", Namespace: "ci"}, nil
}

func (fakePipelineExecutor) ReadRunStatus(context.Context, RunRef) (*RunStatus, error) {
	return &RunStatus{Phase: RunPhaseSucceeded}, nil
}

func (fakePipelineExecutor) CancelRun(context.Context, RunRef) error {
	return nil
}

func TestPipelineExecutorInterface(t *testing.T) {
	var executor PipelineExecutor = fakePipelineExecutor{}
	ref, err := executor.EnsureRun(context.Background(), EnsureRunRequest{})
	if err != nil {
		t.Fatalf("EnsureRun() error = %v", err)
	}
	status, err := executor.ReadRunStatus(context.Background(), *ref)
	if err != nil {
		t.Fatalf("ReadRunStatus() error = %v", err)
	}
	if status.Phase != RunPhaseSucceeded {
		t.Fatalf("phase = %q, want Succeeded", status.Phase)
	}
}

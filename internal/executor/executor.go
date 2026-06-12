package executor

import (
	"context"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

type PipelineExecutor interface {
	EnsureRun(ctx context.Context, req EnsureRunRequest) (*RunRef, error)
	ReadRunStatus(ctx context.Context, ref RunRef) (*RunStatus, error)
	CancelRun(ctx context.Context, ref RunRef) error
}

type EnsureRunRequest struct {
	BuildRun   *cicdv1alpha1.BuildRun
	Project    *cicdv1alpha1.Project
	Repository *cicdv1alpha1.Repository
	Template   *cicdv1alpha1.PipelineTemplate
}

type RunRef struct {
	Kind      string
	Name      string
	Namespace string
}

type RunPhase string

const (
	RunPhasePending   RunPhase = "Pending"
	RunPhaseRunning   RunPhase = "Running"
	RunPhaseSucceeded RunPhase = "Succeeded"
	RunPhaseFailed    RunPhase = "Failed"
	RunPhaseCancelled RunPhase = "Cancelled"
)

type RunStatus struct {
	Phase   RunPhase
	Failure Failure
}

type Failure struct {
	Reason  string
	Message string
}

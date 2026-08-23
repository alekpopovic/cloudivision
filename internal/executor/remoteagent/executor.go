// Package remoteagent contains experimental contracts for a future pull-based
// runner. It does not execute or assign builds in this release.
package remoteagent

import (
	"context"
	"errors"

	"github.com/cloudivision/cloudivision/internal/executor"
)

var ErrExperimentalDisabled = errors.New("remote-agent executor is experimental and disabled")

type PoolType string

const (
	PoolTypeKubernetesJob PoolType = "kubernetes-job"
	PoolTypeRemoteAgent   PoolType = "remote-agent"
)

type RunnerPool struct {
	Name              string
	Type              PoolType
	Labels            map[string]string
	MaxConcurrentRuns int
	ActiveRuns        int
}

type Registration struct {
	RunnerID      string
	Pool          string
	Labels        map[string]string
	TokenID       string // identifier only; raw registration tokens must never be persisted
	ProjectScopes []string
}

type Assignment struct {
	ID         string
	BuildRun   string
	Project    string
	Repository string
	ExpiresAt  int64
}

type Coordinator interface {
	Register(context.Context, Registration) error
	Poll(context.Context, string) (*Assignment, error)
	Complete(context.Context, string, executor.RunStatus) error
}

// Executor is deliberately non-functional. Keeping it outside the controller's
// executor map prevents accidental remote scheduling.
type Executor struct{}

func (Executor) EnsureRun(context.Context, executor.EnsureRunRequest) (*executor.RunRef, error) {
	return nil, ErrExperimentalDisabled
}

func (Executor) ReadRunStatus(context.Context, executor.RunRef) (*executor.RunStatus, error) {
	return nil, ErrExperimentalDisabled
}

func (Executor) CancelRun(context.Context, executor.RunRef) error { return ErrExperimentalDisabled }

func Matches(pool RunnerPool, required map[string]string, project string, registration Registration) bool {
	if pool.Type != PoolTypeRemoteAgent || pool.MaxConcurrentRuns < 1 || pool.ActiveRuns >= pool.MaxConcurrentRuns || pool.Name != registration.Pool {
		return false
	}
	if !contains(registration.ProjectScopes, project) {
		return false
	}
	for key, value := range required {
		if pool.Labels[key] != value || registration.Labels[key] != value {
			return false
		}
	}
	return true
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

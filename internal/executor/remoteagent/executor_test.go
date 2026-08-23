package remoteagent

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudivision/cloudivision/internal/executor"
)

func TestExecutorIsDisabled(t *testing.T) {
	_, err := (Executor{}).EnsureRun(context.Background(), executor.EnsureRunRequest{})
	if !errors.Is(err, ErrExperimentalDisabled) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunnerPoolMatchingRequiresCapacityLabelsAndProject(t *testing.T) {
	pool := RunnerPool{Name: "isolated", Type: PoolTypeRemoteAgent, Labels: map[string]string{"arch": "arm64"}, MaxConcurrentRuns: 2, ActiveRuns: 1}
	registration := Registration{RunnerID: "runner-1", Pool: "isolated", Labels: map[string]string{"arch": "arm64"}, ProjectScopes: []string{"project-a"}}
	if !Matches(pool, map[string]string{"arch": "arm64"}, "project-a", registration) {
		t.Fatal("eligible runner did not match")
	}
	if Matches(pool, map[string]string{"arch": "amd64"}, "project-a", registration) {
		t.Fatal("mismatched label was accepted")
	}
	if Matches(pool, map[string]string{"arch": "arm64"}, "project-b", registration) {
		t.Fatal("unauthorized project was accepted")
	}
	pool.ActiveRuns = 2
	if Matches(pool, map[string]string{"arch": "arm64"}, "project-a", registration) {
		t.Fatal("pool at capacity was accepted")
	}
}

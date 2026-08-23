package logstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// KubernetesStore is the default, non-persistent adapter over runner pod logs.
// Append and Delete are intentionally no-ops because Kubernetes owns pod logs.
type KubernetesStore struct{ Client kubernetes.Interface }

func (s KubernetesStore) Append(context.Context, AppendLogRequest) error { return nil }

func (s KubernetesStore) Read(ctx context.Context, req ReadLogRequest) (*ReadLogResult, error) {
	if s.Client == nil {
		return nil, errors.New("Kubernetes client is required")
	}
	pods, err := s.Client.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{LabelSelector: "cloudivision.io/buildrun=" + req.BuildRun})
	if err != nil {
		return nil, fmt.Errorf("list runner pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, ErrNotFound
	}
	var tail *int64
	if req.TailLines > 0 {
		value := int64(req.TailLines)
		tail = &value
	}
	data, err := s.Client.CoreV1().Pods(req.Namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{TailLines: tail, Timestamps: true}).DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("read runner pod logs: %w", err)
	}
	lines := []LogLine{}
	for _, message := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if message != "" {
			lines = append(lines, LogLine{Message: message})
		}
	}
	return &ReadLogResult{Backend: "kubernetes-pod-logs", Ref: req.Namespace + "/" + pods.Items[0].Name, Lines: FilterAndTail(lines, req.Step, req.TailLines)}, nil
}

func (s KubernetesStore) Stream(ctx context.Context, req StreamLogRequest) (<-chan LogLine, error) {
	result, err := s.Read(ctx, ReadLogRequest{Namespace: req.Namespace, BuildRun: req.BuildRun, TailLines: req.TailLines, Step: req.Step})
	if err != nil {
		return nil, err
	}
	return StreamSnapshot(ctx, result), nil
}

func (KubernetesStore) Delete(context.Context, DeleteLogRequest) error { return nil }

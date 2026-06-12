package api

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type PodLogReader interface {
	Logs(ctx context.Context, namespace, podName string, tailLines *int64) ([]byte, error)
}

type KubernetesPodLogReader struct {
	Client kubernetes.Interface
}

func (r KubernetesPodLogReader) Logs(ctx context.Context, namespace, podName string, tailLines *int64) ([]byte, error) {
	return r.Client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		TailLines: tailLines,
	}).DoRaw(ctx)
}

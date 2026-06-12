package main

import (
	"context"
	"log/slog"
	"os"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/runner"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := runner.ConfigFromEnv(os.Getenv)
	if err != nil {
		logger.Error("invalid runner configuration", "error", err)
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		logger.Error("add Kubernetes scheme", "error", err)
		os.Exit(1)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		logger.Error("add cloudivision scheme", "error", err)
		os.Exit(1)
	}

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		logger.Error("load in-cluster Kubernetes config", "error", err)
		os.Exit(1)
	}
	k8sClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("create Kubernetes client", "error", err)
		os.Exit(1)
	}

	logger.Info("starting cloudivision build runner", "buildRun", cfg.BuildRunName, "namespace", cfg.BuildRunNamespace)
	if err := runner.New(k8sClient, logger).Run(context.Background(), cfg); err != nil {
		logger.Error("runner failed", "error", err)
		os.Exit(1)
	}
	logger.Info("runner completed", "buildRun", cfg.BuildRunName)
}

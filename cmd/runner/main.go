package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/artifacts"
	dependencycache "github.com/cloudivision/cloudivision/internal/cache"
	"github.com/cloudivision/cloudivision/internal/logstore"
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

	buildRunner := runner.New(k8sClient, logger)
	switch strings.ToLower(cfg.LogBackend) {
	case "", "kubernetes-pod-logs":
	case "local":
		buildRunner.LogStore = logstore.LocalStore{Root: cfg.LogRoot}
	case "object":
		buildRunner.LogStore = logstore.ObjectStore{}
	case "loki":
		buildRunner.LogStore = logstore.LokiStore{}
	default:
		logger.Error("unsupported log backend", "backend", cfg.LogBackend)
		os.Exit(1)
	}
	switch strings.ToLower(cfg.ArtifactBackend) {
	case "", "disabled", "noop":
	case "local":
		buildRunner.ArtifactStore = artifacts.LocalStore{Root: cfg.ArtifactRoot}
	case "object":
		buildRunner.ArtifactStore = artifacts.ObjectStore{}
	case "oci":
		buildRunner.ArtifactStore = artifacts.OCIStore{}
	default:
		logger.Error("unsupported artifact backend", "backend", cfg.ArtifactBackend)
		os.Exit(1)
	}
	switch strings.ToLower(cfg.CacheBackend) {
	case "", "disabled":
	case "pvc", "local":
		buildRunner.CacheStore = dependencycache.LocalStore{Root: cfg.CacheRoot}
	case "object-storage":
		buildRunner.CacheStore = dependencycache.ObjectStore{}
	default:
		logger.Error("unsupported cache backend", "backend", cfg.CacheBackend)
		os.Exit(1)
	}
	logger.Info("starting cloudivision build runner", "buildRun", cfg.BuildRunName, "namespace", cfg.BuildRunNamespace)
	if err := buildRunner.Run(context.Background(), cfg); err != nil {
		logger.Error("runner failed", "error", err)
		os.Exit(1)
	}
	logger.Info("runner completed", "buildRun", cfg.BuildRunName)
}

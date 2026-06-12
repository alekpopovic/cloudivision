package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	cloudivisionapi "github.com/cloudivision/cloudivision/internal/api"
	"github.com/cloudivision/cloudivision/internal/audit"
	_ "github.com/jackc/pgx/v5/stdlib"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		logger.Error("add Kubernetes scheme", "error", err)
		os.Exit(1)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		logger.Error("add cloudivision scheme", "error", err)
		os.Exit(1)
	}

	restConfig, err := kubernetesConfig()
	if err != nil {
		logger.Error("load Kubernetes config", "error", err)
		os.Exit(1)
	}
	k8sClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		logger.Error("create Kubernetes API client", "error", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logger.Error("create Kubernetes clientset", "error", err)
		os.Exit(1)
	}
	auditRecorder, auditEvents, webhookIndex, closeAudit, err := configureAudit(context.Background(), logger)
	if err != nil {
		logger.Error("configure audit backend", "error", err)
		os.Exit(1)
	}
	defer closeAudit()

	apiServer := cloudivisionapi.Server{
		Client:           k8sClient,
		LogReader:        cloudivisionapi.KubernetesPodLogReader{Client: clientset},
		Logger:           logger,
		Audit:            auditRecorder,
		AuditEvents:      auditEvents,
		WebhookIndex:     webhookIndex,
		DefaultNamespace: envOrDefault("CLOU_DIVISION_DEFAULT_NAMESPACE", "default"),
		AuthMode:         envOrDefault("CLOU_DIVISION_AUTH_MODE", "disabled"),
		CORSOrigins:      csvEnv("CLOU_DIVISION_CORS_ALLOWED_ORIGINS", "http://localhost:4200,http://localhost:4201"),
		MetricsEnabled:   envBool("CLOU_DIVISION_METRICS_ENABLED", true),
	}

	addr := envOrDefault("CLOU_DIVISION_API_ADDR", envOrDefault("CLOUDIVISION_API_ADDR", ":8080"))
	server := &http.Server{
		Addr:              addr,
		Handler:           apiServer.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting cloudivision API server", "addr", addr, "authMode", apiServer.AuthMode)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("api server stopped")
}

func kubernetesConfig() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func csvEnv(name, fallback string) []string {
	value := envOrDefault(name, fallback)
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func configureAudit(ctx context.Context, logger *slog.Logger) (audit.Recorder, audit.EventLister, audit.WebhookIndexer, func(), error) {
	backend := strings.ToLower(envOrDefault("CLOU_DIVISION_AUDIT_BACKEND", "log"))
	databaseURL := os.Getenv("CLOU_DIVISION_DATABASE_URL")
	switch backend {
	case "noop":
		return audit.NoopRecorder{}, nil, nil, func() {}, nil
	case "log", "":
		return audit.LogRecorder{Logger: logger}, nil, nil, func() {}, nil
	case "postgres":
		if databaseURL == "" {
			return nil, nil, nil, func() {}, fmt.Errorf("CLOU_DIVISION_DATABASE_URL is required when CLOU_DIVISION_AUDIT_BACKEND=postgres")
		}
		db, err := sql.Open("pgx", databaseURL)
		if err != nil {
			return nil, nil, nil, func() {}, fmt.Errorf("open postgres audit database: %w", err)
		}
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			_ = db.Close()
			return nil, nil, nil, func() {}, fmt.Errorf("ping postgres audit database: %w", err)
		}
		recorder := audit.NewPostgresRecorder(db)
		return recorder, recorder, recorder, func() { _ = db.Close() }, nil
	default:
		return nil, nil, nil, func() {}, fmt.Errorf("unsupported CLOU_DIVISION_AUDIT_BACKEND %q", backend)
	}
}

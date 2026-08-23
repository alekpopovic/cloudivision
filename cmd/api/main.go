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
	"github.com/cloudivision/cloudivision/internal/artifacts"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/auth"
	dependencycache "github.com/cloudivision/cloudivision/internal/cache"
	"github.com/cloudivision/cloudivision/internal/logstore"
	"github.com/cloudivision/cloudivision/internal/policy"
	"github.com/cloudivision/cloudivision/internal/provider"
	providerbuild "github.com/cloudivision/cloudivision/internal/provider/build"
	providergit "github.com/cloudivision/cloudivision/internal/provider/git"
	providergitops "github.com/cloudivision/cloudivision/internal/provider/gitops"
	providernotifications "github.com/cloudivision/cloudivision/internal/provider/notifications"
	providerregistry "github.com/cloudivision/cloudivision/internal/provider/registry"
	providersecrets "github.com/cloudivision/cloudivision/internal/provider/secrets"
	providersupplychain "github.com/cloudivision/cloudivision/internal/provider/supplychain"
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
	authMode := envOrDefault("CLOU_DIVISION_AUTH_MODE", "disabled")
	authenticator, err := configureAuthenticator(authMode)
	if err != nil {
		logger.Error("configure auth", "error", err)
		os.Exit(1)
	}
	providerRegistry, err := configureProviderRegistry()
	if err != nil {
		logger.Error("configure provider registry", "error", err)
		os.Exit(1)
	}
	configuredLogStore, err := configureLogStore()
	if err != nil {
		logger.Error("configure log backend", "error", err)
		os.Exit(1)
	}
	configuredArtifactStore, err := configureArtifactStore()
	if err != nil {
		logger.Error("configure artifact backend", "error", err)
		os.Exit(1)
	}
	configuredCacheStore, err := configureCacheStore()
	if err != nil {
		logger.Error("configure dependency cache backend", "error", err)
		os.Exit(1)
	}

	apiServer := cloudivisionapi.Server{
		Client:           k8sClient,
		LogReader:        cloudivisionapi.KubernetesPodLogReader{Client: clientset},
		LogStore:         configuredLogStore,
		ArtifactStore:    configuredArtifactStore,
		CacheStore:       configuredCacheStore,
		Logger:           logger,
		Audit:            auditRecorder,
		AuditEvents:      auditEvents,
		WebhookIndex:     webhookIndex,
		Authenticator:    authenticator,
		DefaultNamespace: envOrDefault("CLOU_DIVISION_DEFAULT_NAMESPACE", "default"),
		AuthMode:         authMode,
		CORSOrigins:      csvEnv("CLOU_DIVISION_CORS_ALLOWED_ORIGINS", "http://localhost:4200,http://localhost:4201"),
		MetricsEnabled:   envBool("CLOU_DIVISION_METRICS_ENABLED", true),
		Providers:        providerRegistry,
		PolicyEvaluator:  policy.NewDefaultEvaluator(),
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

func configureLogStore() (logstore.LogStore, error) {
	switch strings.ToLower(envOrDefault("CLOU_DIVISION_LOG_BACKEND", "kubernetes-pod-logs")) {
	case "", "kubernetes-pod-logs":
		return nil, nil
	case "local":
		root := os.Getenv("CLOU_DIVISION_LOG_ROOT")
		if root == "" {
			return nil, errors.New("CLOU_DIVISION_LOG_ROOT is required for the local log backend")
		}
		return logstore.LocalStore{Root: root}, nil
	case "object":
		return logstore.ObjectStore{}, nil
	case "loki":
		return logstore.LokiStore{}, nil
	default:
		return nil, fmt.Errorf("unsupported CLOU_DIVISION_LOG_BACKEND")
	}
}

func configureArtifactStore() (artifacts.ArtifactStore, error) {
	switch strings.ToLower(envOrDefault("CLOU_DIVISION_ARTIFACT_BACKEND", "disabled")) {
	case "", "disabled", "noop":
		return nil, nil
	case "local":
		root := os.Getenv("CLOU_DIVISION_ARTIFACT_ROOT")
		if root == "" {
			return nil, errors.New("CLOU_DIVISION_ARTIFACT_ROOT is required for the local artifact backend")
		}
		return artifacts.LocalStore{Root: root}, nil
	case "object":
		return artifacts.ObjectStore{}, nil
	case "oci":
		return artifacts.OCIStore{}, nil
	default:
		return nil, fmt.Errorf("unsupported CLOU_DIVISION_ARTIFACT_BACKEND")
	}
}

func configureCacheStore() (dependencycache.Store, error) {
	root := os.Getenv("CLOU_DIVISION_CACHE_ROOT")
	if root == "" || os.Getenv("CLOU_DIVISION_CACHE_PVC") == "" {
		return nil, nil
	}
	return dependencycache.LocalStore{Root: root}, nil
}

func configureProviderRegistry() (*provider.Registry, error) {
	registry := provider.NewRegistry()
	providers := []provider.Provider{
		providergit.Generic(), providergit.GitHub(), providergit.GitLab(),
		providerregistry.Generic(), providerregistry.GHCR(), providerregistry.GitLab(), providerregistry.Harbor(),
		providerregistry.ECR(), providerregistry.GCR(), providerregistry.ACR(), providersecrets.Kubernetes(),
		providergitops.Generic(), providergitops.ArgoCD(), providerbuild.BuildKit(),
		providernotifications.Noop(), providersupplychain.Noop(), providersupplychain.Syft(),
		providersupplychain.Grype(), providersupplychain.Cosign(),
	}
	for _, current := range providers {
		if err := registry.Register(current); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func configureAuthenticator(mode string) (auth.Authenticator, error) {
	switch strings.ToLower(mode) {
	case "disabled", "":
		return auth.DisabledAuthenticator{}, nil
	case "oidc":
		groups, err := auth.LoadGroupMappingFile(os.Getenv("CLOU_DIVISION_AUTH_GROUPS_FILE"))
		if err != nil {
			return nil, err
		}
		return auth.NewOIDCAuthenticator(auth.OIDCConfig{
			IssuerURL: os.Getenv("CLOU_DIVISION_OIDC_ISSUER_URL"),
			ClientID:  os.Getenv("CLOU_DIVISION_OIDC_CLIENT_ID"),
			Audience:  os.Getenv("CLOU_DIVISION_OIDC_AUDIENCE"),
			JWKSURL:   os.Getenv("CLOU_DIVISION_OIDC_JWKS_URL"),
			Groups:    groups,
		})
	default:
		return nil, fmt.Errorf("unsupported CLOU_DIVISION_AUTH_MODE %q", mode)
	}
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

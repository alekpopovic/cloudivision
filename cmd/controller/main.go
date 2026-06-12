package main

import (
	"os"
	"strconv"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	cloudivisioncontroller "github.com/cloudivision/cloudivision/internal/controller"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func main() {
	scheme := runtime.NewScheme()
	ctrl.SetLogger(zap.New(zap.UseDevMode(envBool("CLOUDIVISION_CONTROLLER_DEV_LOGS", false))))

	setupLog := ctrl.Log.WithName("setup")

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "unable to add Kubernetes scheme")
		os.Exit(1)
	}
	if err := cicdv1alpha1.AddToScheme(scheme); err != nil {
		setupLog.Error(err, "unable to add cloudivision scheme")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: envOrDefault("CLOUDIVISION_METRICS_BIND_ADDRESS", ":8080"),
		},
		HealthProbeBindAddress: envOrDefault("CLOUDIVISION_HEALTH_PROBE_BIND_ADDRESS", ":8081"),
		LeaderElection:         envBool("CLOUDIVISION_LEADER_ELECTION", false),
		LeaderElectionID:       "cloudivision-controller.cicd.cloudivision.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := (&cloudivisioncontroller.ProjectReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Project controller")
		os.Exit(1)
	}
	if err := (&cloudivisioncontroller.BuildRunReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create BuildRun controller")
		os.Exit(1)
	}
	if err := (&cloudivisioncontroller.ReleaseReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create Release controller")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting cloudivision controller manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "controller manager stopped")
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
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

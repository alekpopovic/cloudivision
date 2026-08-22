package observability

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registerOnce sync.Once

	BuildRunTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_buildrun_total",
		Help: "BuildRun status updates observed by phase.",
	}, []string{"phase"})

	BuildRunDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "cloudivision_buildrun_duration_seconds",
		Help:    "BuildRun duration from start to completion.",
		Buckets: prometheus.DefBuckets,
	})

	BuildQueueDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "cloudivision_buildrun_queue_duration_seconds",
		Help:    "Time from BuildRun creation until the executor reports Running.",
		Buckets: prometheus.DefBuckets,
	})

	RunnerJobFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cloudivision_runner_job_failures_total",
		Help: "Job executor failures observed by the BuildRun controller.",
	})

	ReleaseTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_release_status_updates_total",
		Help: "Release status updates observed by phase; this is a transition/update counter, not a current-object gauge.",
	}, []string{"phase"})

	ReleaseDeploymentDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "cloudivision_release_deployment_duration_seconds",
		Help:    "Release duration from GitOps preparation start until a terminal result.",
		Buckets: []float64{5, 15, 30, 60, 120, 300, 600, 1200, 1800, 3600},
	})

	ReleaseInProgressStarted = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cloudivision_release_in_progress_started_timestamp_seconds",
		Help: "Start timestamp for an in-progress Release; namespace/name labels are retained only while actionable.",
	}, []string{"namespace", "name", "phase"})

	GitOpsFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_gitops_failures_total",
		Help: "GitOps operation failures by bounded operation name.",
	}, []string{"operation"})

	WebhookEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_webhook_events_total",
		Help: "Webhook outcomes by bounded provider and outcome values.",
	}, []string{"provider", "outcome"})

	AuditWriteFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cloudivision_audit_write_failures_total",
		Help: "Audit and webhook-index persistence failures.",
	})

	ReconcileErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_reconcile_errors_total",
		Help: "Controller reconcile errors by controller name.",
	}, []string{"controller"})

	ReconcileDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cloudivision_reconcile_duration_seconds",
		Help:    "Controller reconcile duration by controller name.",
		Buckets: prometheus.DefBuckets,
	}, []string{"controller"})

	HTTPRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_http_requests_total",
		Help: "API HTTP requests by method, route, and status.",
	}, []string{"method", "route", "status"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cloudivision_http_request_duration_seconds",
		Help:    "API HTTP request latency by method and route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	HTTPErrorTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cloudivision_http_errors_total",
		Help: "API HTTP error responses by method, route, and status.",
	}, []string{"method", "route", "status"})
)

func RegisterMetrics() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			BuildRunTotal,
			BuildRunDuration,
			BuildQueueDuration,
			RunnerJobFailures,
			ReleaseTotal,
			ReleaseDeploymentDuration,
			ReleaseInProgressStarted,
			GitOpsFailures,
			WebhookEvents,
			AuditWriteFailures,
			ReconcileErrors,
			ReconcileDuration,
			HTTPRequestTotal,
			HTTPRequestDuration,
			HTTPErrorTotal,
		)
	})
}

var inProgressReleasePhases = []string{"PreparingGitOpsChange", "GitOpsChangeCommitted", "WaitingForSync"}

func ObserveReleaseInProgress(namespace, name, phase string, startedAt *time.Time) {
	RegisterMetrics()
	ClearReleaseInProgress(namespace, name)
	if startedAt == nil {
		return
	}
	for _, candidate := range inProgressReleasePhases {
		if phase == candidate {
			ReleaseInProgressStarted.WithLabelValues(namespace, name, phase).Set(float64(startedAt.Unix()))
			return
		}
	}
}

func ClearReleaseInProgress(namespace, name string) {
	for _, phase := range inProgressReleasePhases {
		ReleaseInProgressStarted.DeleteLabelValues(namespace, name, phase)
	}
}

func MetricsHandler() http.Handler {
	RegisterMetrics()
	return promhttp.Handler()
}

func ObserveReconcile(controller string, started time.Time, err error) {
	RegisterMetrics()
	ReconcileDuration.WithLabelValues(controller).Observe(time.Since(started).Seconds())
	if err != nil {
		ReconcileErrors.WithLabelValues(controller).Inc()
	}
}

func ObserveHTTPRequest(method, route string, status int, started time.Time) {
	RegisterMetrics()
	statusLabel := strconv.Itoa(status)
	HTTPRequestTotal.WithLabelValues(method, route, statusLabel).Inc()
	HTTPRequestDuration.WithLabelValues(method, route).Observe(time.Since(started).Seconds())
	if status >= 400 {
		HTTPErrorTotal.WithLabelValues(method, route, statusLabel).Inc()
	}
}

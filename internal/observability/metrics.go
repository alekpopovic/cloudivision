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
			ReconcileErrors,
			ReconcileDuration,
			HTTPRequestTotal,
			HTTPRequestDuration,
			HTTPErrorTotal,
		)
	})
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

package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/topfreegames/mqtt-history/logger"
)

// ResponseTimeSeconds is the histogram of HTTP request durations in seconds.
// It is registered with the default Prometheus registry at package init time
// (NOT inside App.Configure) so that the many App instances created during
// tests do not panic with a duplicate-registration error.
//
// Buckets use prometheus.DefBuckets and can be tuned if request latencies
// fall outside the default range.
//
// Note on cardinality: the gameID label is unbounded in principle. Each
// distinct gameID multiplies the series count by (#buckets + 2) per
// route/method/status combination. This is acceptable while the active set of
// games stays small; revisit if Datadog custom-metric counts grow.
var ResponseTimeSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"route", "method", "status", "gameID"},
)

// Prometheus is a thin metrics client: the
// middleware reports through it instead of touching collectors directly, so new
// metrics can be added here without changing the middleware wiring. A nil
// *Prometheus means the Prometheus backend is disabled.
type Prometheus struct {
	responseTime *prometheus.HistogramVec
}

// NewPrometheus returns a Prometheus metrics client backed by the package-level
// collectors (registered once at init to stay test-safe).
func NewPrometheus() *Prometheus {
	return &Prometheus{responseTime: ResponseTimeSeconds}
}

// Timing observes an HTTP request duration (in seconds) in the response-time
// histogram.
func (p *Prometheus) Timing(value time.Duration, route, method, status, gameID string) {
	p.responseTime.WithLabelValues(route, method, status, gameID).Observe(value.Seconds())
}

// startMetricsServer exposes the Prometheus /metrics endpoint on a dedicated
// internal HTTP server, separate from the public Echo API. Company policy
// requires that /metrics is never served by the public application controller.
func startMetricsServer(port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	logger.Logger.Infof("Starting Prometheus metrics server on %s", addr)
	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			logger.Logger.Errorf("Metrics server stopped: %v", err)
		}
	}()
}

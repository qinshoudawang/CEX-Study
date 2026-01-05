package httpinfra

import "github.com/prometheus/client_golang/prometheus"

const namespace = "withdraw"

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_latency_seconds",
			Help:      "HTTP request latency",
			Buckets:   []float64{0.05, 0.1, 0.2, 0.5, 1, 2, 5},
		},
		[]string{"method", "path"},
	)
)

func Register() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestLatency,
	)
}

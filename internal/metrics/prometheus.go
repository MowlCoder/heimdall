package metrics

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusCollector struct {
	RequestTotal    *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{
		RequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_client_requests_total",
				Help: "A counter for requests made by the client.",
			},
			[]string{"code", "method"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_client_request_duration_seconds",
				Help:    "A histogram of request latencies.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		)}
}

func (pc *PrometheusCollector) StartServer(port string) {

	go func() {

		http.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Prometheus metrics server failed: %v", err)
		}

	}()

}

func (pc *PrometheusCollector) IncRequestsTotal(labels MetricLabels) {

	pc.RequestTotal.WithLabelValues(
		labels["code"],
		labels["method"],
	).Inc()

}

func (pc *PrometheusCollector) ObserveRequestDuration(duration time.Duration, labels MetricLabels) {
	pc.RequestDuration.WithLabelValues(
		labels["method"],
	).Observe(duration.Seconds())
}

package metrics

import "time"

type MetricLabels map[string]string

type MetricsCollector interface {
	IncRequestsTotal(labels MetricLabels)

	ObserveRequestDuration(duration time.Duration, labels MetricLabels)
}

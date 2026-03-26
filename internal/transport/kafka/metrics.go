package kafka

import basemetrics "casino-transaction-system/internal/observability/metrics"

type metricLabels = basemetrics.Labels

// MetricsSink is an infrastructure abstraction for consumer metrics collection.
// It allows switching from log-based metrics to Prometheus/OpenTelemetry without
// changing consumer business flow.
type MetricsSink = basemetrics.Sink

func newLogMetricsSink() MetricsSink {
	return basemetrics.NewLogSink()
}

func metricKey(name string, labels metricLabels) string {
	return basemetrics.Key(name, labels)
}

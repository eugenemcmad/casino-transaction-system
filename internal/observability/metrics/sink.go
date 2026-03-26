package metrics

import (
	"sort"
	"strings"
	"time"
)

// Labels describe metric dimensions (tags).
type Labels map[string]string

// Sink is a transport-agnostic metrics contract.
// It can be reused by Kafka, PostgreSQL, HTTP/API and other adapters.
//
// Prometheus-ready note:
// This interface already matches common Prometheus usage patterns
// (counter, gauge, duration observation). Current implementation can be log-based,
// but switching to Prometheus should only require a new Sink implementation.
//
// Prometheus connection plan:
// 1) Implement Sink using Prometheus client collectors (CounterVec/GaugeVec/HistogramVec).
// 2) Keep metric names and labels consistent with current keys.
// 3) Wire the new implementation in bootstrap instead of NewLogSink().
// 4) Expose /metrics endpoint (promhttp.Handler()) in HTTP router.
type Sink interface {
	IncCounter(name string, labels Labels, value int64)
	SetGauge(name string, labels Labels, value float64)
	ObserveDuration(name string, labels Labels, d time.Duration)
	Flush()
}

// Key builds a stable metric key from name + labels.
func Key(name string, labels Labels) string {
	if len(labels) == 0 {
		return name
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(name)
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(labels[k])
	}
	b.WriteString("}")
	return b.String()
}

package metrics

import (
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"
)

type timerWindow struct {
	values []float64
}

// LogSink is a simple in-memory sink that periodically flushes metrics to logs.
// It is the current default transport for local/debug visibility.
//
// Production note:
// this sink is intentionally kept behind the Sink interface so it can be replaced
// by a Prometheus/OpenTelemetry implementation without changing business flows.
type LogSink struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
	timers   map[string]*timerWindow
}

func NewLogSink() *LogSink {
	return &LogSink{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
		timers:   make(map[string]*timerWindow),
	}
}

func (m *LogSink) IncCounter(name string, labels Labels, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := Key(name, labels)
	m.counters[key] += value
}

func (m *LogSink) SetGauge(name string, labels Labels, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := Key(name, labels)
	m.gauges[key] = value
}

func (m *LogSink) ObserveDuration(name string, labels Labels, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := Key(name, labels)
	tw, ok := m.timers[key]
	if !ok {
		tw = &timerWindow{}
		m.timers[key] = tw
	}
	tw.values = append(tw.values, float64(d.Milliseconds()))
}

func (m *LogSink) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.counters) == 0 && len(m.gauges) == 0 && len(m.timers) == 0 {
		return
	}

	timers := make(map[string]map[string]float64, len(m.timers))
	for key, win := range m.timers {
		if len(win.values) == 0 {
			continue
		}
		timers[key] = durationStats(win.values)
	}

	slog.Info("metrics_flush",
		"counters", cloneCounters(m.counters),
		"gauges", cloneGauges(m.gauges),
		"timers_ms", timers,
	)

	clear(m.counters)
	clear(m.gauges)
	clear(m.timers)
}

func durationStats(values []float64) map[string]float64 {
	sort.Float64s(values)

	count := float64(len(values))
	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return map[string]float64{
		"count": count,
		"avg":   sum / count,
		"p50":   quantile(values, 0.50),
		"p95":   quantile(values, 0.95),
		"max":   values[len(values)-1],
	}
}

func cloneCounters(src map[string]int64) map[string]int64 {
	dst := make(map[string]int64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneGauges(src map[string]float64) map[string]float64 {
	dst := make(map[string]float64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[len(sorted)-1]
	}

	pos := q * float64(len(sorted)-1)
	left := int(math.Floor(pos))
	right := int(math.Ceil(pos))
	if left == right {
		return sorted[left]
	}
	weight := pos - float64(left)
	return sorted[left]*(1-weight) + sorted[right]*weight
}

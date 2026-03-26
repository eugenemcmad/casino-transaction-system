package metrics

import (
	"testing"
	"time"
)

func TestKey_SortsLabelsDeterministically(t *testing.T) {
	key := Key("kafka_consumer_messages_total", Labels{
		"partition": "3",
		"topic":     "transactions",
		"result":    "processed",
	})
	want := "kafka_consumer_messages_total{partition=3,result=processed,topic=transactions}"
	if key != want {
		t.Fatalf("Key() = %q, want %q", key, want)
	}
}

func TestLogSink_FlushClearsWindow(t *testing.T) {
	sink := NewLogSink()
	sink.IncCounter("counter", Labels{"result": "ok"}, 2)
	sink.SetGauge("gauge", Labels{"partition": "0"}, 12.5)
	sink.ObserveDuration("timer", Labels{"result": "ok"}, 20*time.Millisecond)
	sink.ObserveDuration("timer", Labels{"result": "ok"}, 40*time.Millisecond)

	sink.Flush()

	if len(sink.counters) != 0 {
		t.Fatalf("counters not cleared after flush: %v", sink.counters)
	}
	if len(sink.gauges) != 0 {
		t.Fatalf("gauges not cleared after flush: %v", sink.gauges)
	}
	if len(sink.timers) != 0 {
		t.Fatalf("timers not cleared after flush: %v", sink.timers)
	}
}

func TestDurationStats_ComputesPercentiles(t *testing.T) {
	stats := durationStats([]float64{10, 20, 30, 40, 50})

	if stats["count"] != 5 {
		t.Fatalf("count = %v, want 5", stats["count"])
	}
	if stats["avg"] != 30 {
		t.Fatalf("avg = %v, want 30", stats["avg"])
	}
	if stats["p50"] != 30 {
		t.Fatalf("p50 = %v, want 30", stats["p50"])
	}
	if stats["p95"] != 48 {
		t.Fatalf("p95 = %v, want 48", stats["p95"])
	}
	if stats["max"] != 50 {
		t.Fatalf("max = %v, want 50", stats["max"])
	}
}

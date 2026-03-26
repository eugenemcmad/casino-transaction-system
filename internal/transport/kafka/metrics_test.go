package kafka

import (
	"testing"
)

func TestMetricKey_SortsLabelsDeterministically(t *testing.T) {
	key := metricKey("kafka_consumer_messages_total", metricLabels{
		"partition": "3",
		"topic":     "transactions",
		"result":    "processed",
	})
	want := "kafka_consumer_messages_total{partition=3,result=processed,topic=transactions}"
	if key != want {
		t.Fatalf("metricKey() = %q, want %q", key, want)
	}
}

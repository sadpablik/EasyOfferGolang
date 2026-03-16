package events

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var kafkaPublishTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "question",
		Name:      "kafka_publish_total",
		Help:      "Kafka write attempts by event type and status.",
	},
	[]string{"event_type", "status"},
)

var registerKafkaMetricsOnce sync.Once

func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	registerKafkaMetricsOnce.Do(func() {
		reg.MustRegister(kafkaPublishTotal)
	})
}

func ObservePublish(eventType string, err error) {
	status := "success"
	if err != nil {
		status = "fail"
	}
	kafkaPublishTotal.WithLabelValues(eventType, status).Inc()
}

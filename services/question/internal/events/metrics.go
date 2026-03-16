package events

import (
	"easyoffer/question/internal/domain"
	"strings"
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

var outboxDispatchTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "question",
		Name:      "outbox_dispatch_total",
		Help:      "Outbox dispatch attempts by event type and status.",
	},
	[]string{"event_type", "status"},
)

var outboxQueueSize = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "easyoffer",
		Subsystem: "question",
		Name:      "outbox_queue_size",
		Help:      "Current number of outbox events by status.",
	},
	[]string{"status"},
)

var trackedOutboxStatuses = []domain.OutboxStatus{
	domain.OutboxStatusPending,
	domain.OutboxStatusFailed,
	domain.OutboxStatusSent,
}

var registerKafkaMetricsOnce sync.Once

func RegisterMetrics(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	registerKafkaMetricsOnce.Do(func() {
		reg.MustRegister(kafkaPublishTotal, outboxDispatchTotal, outboxQueueSize)
		ObserveOutboxQueueSize(nil)
	})
}

func ObservePublish(eventType string, err error) {
	status := "success"
	if err != nil {
		status = "fail"
	}
	kafkaPublishTotal.WithLabelValues(eventType, status).Inc()
}

func ObserveOutboxDispatch(eventType, status string) {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		eventType = "unknown"
	}

	status = strings.TrimSpace(status)
	if status == "" {
		status = "unknown"
	}

	outboxDispatchTotal.WithLabelValues(eventType, status).Inc()
}

func ObserveOutboxQueueSize(counts map[domain.OutboxStatus]int64) {
	for _, status := range trackedOutboxStatuses {
		value := int64(0)
		if counts != nil {
			value = counts[status]
		}
		outboxQueueSize.WithLabelValues(string(status)).Set(float64(value))
	}
}

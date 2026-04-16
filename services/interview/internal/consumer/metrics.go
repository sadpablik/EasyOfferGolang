package consumer

import (
    "strings"
    "sync"

    "github.com/prometheus/client_golang/prometheus"
)

var kafkaConsumeTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "easyoffer",
        Subsystem: "interview",
        Name:      "kafka_consume_total",
        Help:      "Kafka consume handling attempts by event type and status.",
    },
    []string{"event_type", "status"},
)

var registerKafkaMetricsOnce sync.Once

func RegisterMetrics(reg prometheus.Registerer) {
    if reg == nil {
        reg = prometheus.DefaultRegisterer
    }

    registerKafkaMetricsOnce.Do(func() {
        reg.MustRegister(kafkaConsumeTotal)
    })
}

func ObserveConsume(eventType string, err error) {
    eventType = strings.TrimSpace(eventType)
    if eventType == "" {
        eventType = "unknown"
    }

    status := "success"
    if err != nil {
        status = "fail"
    }

    kafkaConsumeTotal.WithLabelValues(eventType, status).Inc()
}
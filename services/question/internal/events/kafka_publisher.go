package events

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"easyoffer/question/internal/domain"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokers, topic string) (*KafkaPublisher, error) {
	brokers = strings.TrimSpace(brokers)
	topic = strings.TrimSpace(topic)

	if brokers == "" {
		return nil, errors.New("kafka brokers are required")
	}
	if topic == "" {
		return nil, errors.New("kafka topic is required")
	}

	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokerList...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &KafkaPublisher{writer: w}, nil
}

func (p *KafkaPublisher) PublishQuestionCreated(ctx context.Context, q *domain.Question) error {
	if q == nil {
		return errors.New("question is nil")
	}
	return p.publish(ctx, EventQuestionCreated, payloadFromQuestion(q), q.ID)
}

func (p *KafkaPublisher) PublishQuestionUpdated(ctx context.Context, q *domain.Question) error {
	if q == nil {
		return errors.New("question is nil")
	}
	return p.publish(ctx, EventQuestionUpdated, payloadFromQuestion(q), q.ID)
}

func (p *KafkaPublisher) PublishQuestionDeleted(ctx context.Context, questionID string) error {
	if strings.TrimSpace(questionID) == "" {
		return errors.New("question ID is required")
	}
	payload := QuestionPayload{
		QuestionID: questionID,
	}
	return p.publish(ctx, EventQuestionDeleted, payload, questionID)
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}

func (p *KafkaPublisher) publish(ctx context.Context, eventType string, payload QuestionPayload, key string) error {
	event := QuestionEvent{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		OccurredAt: time.Now().UTC(),
		Version:    1,
		Payload:    payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if strings.TrimSpace(key) == "" {
		key = event.EventID
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  event.OccurredAt,
	})
	ObservePublish(eventType, err)
	return err
}

func (p *KafkaPublisher) PublishOutboxEvent(ctx context.Context, eventType, key string, payload []byte) error {
	if len(payload) == 0 {
		return errors.New("event payload is empty")
	}
	if strings.TrimSpace(key) == "" {
		key = uuid.NewString()
	}

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	})
	ObservePublish(eventType, err)
	return err
}

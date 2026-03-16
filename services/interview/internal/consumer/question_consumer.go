package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"easyoffer/interview/internal/repository"

	"github.com/segmentio/kafka-go"
)

type QuestionConsumer struct {
	reader *kafka.Reader
	repo   repository.QuestionRepository
}

func NewQuestionConsumer(brokers, topic, groupID string, repo repository.QuestionRepository) (*QuestionConsumer, error) {
	brokers = strings.TrimSpace(brokers)
	topic = strings.TrimSpace(topic)
	groupID = strings.TrimSpace(groupID)

	if brokers == "" {
		return nil, errors.New("kafka brokers are required")
	}
	if topic == "" {
		return nil, errors.New("kafka topic is required")
	}
	if groupID == "" {
		groupID = "interview-service"
	}

	brokerList := strings.Split(brokers, ",")
	for i := range brokerList {
		brokerList[i] = strings.TrimSpace(brokerList[i])
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokerList,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset,
	})

	return &QuestionConsumer{
		reader: reader,
		repo:   repo,
	}, nil
}

func (c *QuestionConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		if err := c.handleMessage(ctx, msg.Value); err != nil {
			log.Printf("failed to handle question event topic=%s partition=%d offset=%d: %v",
				msg.Topic, msg.Partition, msg.Offset, err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

func (c *QuestionConsumer) Close() error {
	return c.reader.Close()
}

func (c *QuestionConsumer) handleMessage(ctx context.Context, data []byte) error {
	var event QuestionEvent

	if err := json.Unmarshal(data, &event); err != nil {
		return err
	}

	switch event.EventType {
	case EventQuestionCreated, EventQuestionUpdated:
		return c.repo.Upsert(ctx, event.Payload.ToSnapshot())
	case EventQuestionDeleted:
		return c.repo.DeleteQuestion(ctx, event.Payload.QuestionID)
	default:
		log.Printf("skip unknown question event type=%s", event.EventType)
		return nil
	}
}

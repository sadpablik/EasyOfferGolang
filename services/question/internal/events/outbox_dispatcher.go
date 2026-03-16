package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"easyoffer/question/internal/domain"
)

type OutboxStore interface {
	ListOutboxForDispatch(limit int, now time.Time) ([]*domain.OutboxEvent, error)
	MarkOutboxSent(eventID string, sentAt time.Time) error
	MarkOutboxRetry(eventID string, nextRetryAt time.Time, lastError string) error
	CountOutboxByStatus() (map[domain.OutboxStatus]int64, error)
}

type OutboxPublisher interface {
	PublishOutboxEvent(ctx context.Context, eventType, key string, payload []byte) error
}

type OutboxDispatcher struct {
	store        OutboxStore
	publisher    OutboxPublisher
	batchSize    int
	pollInterval time.Duration
}

func NewOutboxDispatcher(store OutboxStore, publisher OutboxPublisher, batchSize int, pollInterval time.Duration) *OutboxDispatcher {
	if batchSize <= 0 {
		batchSize = 50
	}
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	return &OutboxDispatcher{
		store:        store,
		publisher:    publisher,
		batchSize:    batchSize,
		pollInterval: pollInterval,
	}
}

func (d *OutboxDispatcher) Run(ctx context.Context) {
	// Первый проход сразу при старте, чтобы не ждать тикер.
	d.dispatchOnce(ctx)

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.dispatchOnce(ctx)
		}
	}
}

func (d *OutboxDispatcher) dispatchOnce(ctx context.Context) {
	if d.store == nil || d.publisher == nil {
		log.Printf("outbox: dispatcher dependencies are not configured")
		return
	}

	defer d.refreshQueueSize()

	eventsBatch, err := d.store.ListOutboxForDispatch(d.batchSize, time.Now().UTC())
	if err != nil {
		log.Printf("outbox: failed to load batch: %v", err)
		ObserveOutboxDispatch("batch", "load_fail")
		return
	}

	for _, evt := range eventsBatch {
		if ctx.Err() != nil {
			return
		}
		if evt == nil {
			continue
		}

		payload := strings.TrimSpace(evt.Payload)
		if payload == "" {
			if err := d.markRetry(evt, errors.New("empty outbox payload")); err != nil {
				log.Printf("outbox: failed to mark retry id=%s: %v", evt.ID, err)
				ObserveOutboxDispatch(evt.EventType, "mark_retry_fail")
				continue
			}
			ObserveOutboxDispatch(evt.EventType, "retry")
			continue
		}

		key := strings.TrimSpace(evt.AggregateID)
		if key == "" {
			key = evt.ID
		}

		err := d.publisher.PublishOutboxEvent(ctx, evt.EventType, key, []byte(payload))
		if err == nil {
			if markErr := d.store.MarkOutboxSent(evt.ID, time.Now().UTC()); markErr != nil {
				log.Printf("outbox: failed to mark sent id=%s: %v", evt.ID, markErr)
				ObserveOutboxDispatch(evt.EventType, "mark_sent_fail")
				continue
			}
			ObserveOutboxDispatch(evt.EventType, "sent")
			continue
		}

		if markErr := d.markRetry(evt, err); markErr != nil {
			log.Printf("outbox: failed to mark retry id=%s: %v", evt.ID, markErr)
			ObserveOutboxDispatch(evt.EventType, "mark_retry_fail")
			continue
		}
		ObserveOutboxDispatch(evt.EventType, "retry")
	}
}

func (d *OutboxDispatcher) refreshQueueSize() {
	counts, err := d.store.CountOutboxByStatus()
	if err != nil {
		log.Printf("outbox: failed to refresh queue size metrics: %v", err)
		return
	}
	ObserveOutboxQueueSize(counts)
}

func (d *OutboxDispatcher) markRetry(evt *domain.OutboxEvent, reason error) error {
	if evt == nil {
		return fmt.Errorf("outbox event is nil")
	}

	retryErr := reason
	if retryErr == nil {
		retryErr = errors.New("retry reason is missing")
	}

	nextRetry := time.Now().UTC().Add(retryDelay(evt.Attempts + 1))
	return d.store.MarkOutboxRetry(evt.ID, nextRetry, retryErr.Error())
}

func retryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return time.Second
	}

	delay := time.Second
	for i := 1; i < attempt && delay < 30*time.Second; i++ {
		delay *= 2
	}
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}

package events_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"easyoffer/question/internal/domain"
	events "easyoffer/question/internal/events"
)

type publishCall struct {
	eventType string
	key       string
	payload   string
}

type outboxPublisherStub struct {
	err   error
	calls []publishCall
}

func (s *outboxPublisherStub) PublishOutboxEvent(_ context.Context, eventType, key string, payload []byte) error {
	s.calls = append(s.calls, publishCall{
		eventType: eventType,
		key:       key,
		payload:   string(payload),
	})
	return s.err
}

type retryCall struct {
	eventID     string
	nextRetryAt time.Time
	lastError   string
}

type outboxStoreStub struct {
	events      []*domain.OutboxEvent
	counts      map[domain.OutboxStatus]int64
	listErr     error
	countErr    error
	countCalls  int
	sentIDs     []string
	retryCalls  []retryCall
	lastBatchAt time.Time
}

func newOutboxStoreStub(eventsBatch ...*domain.OutboxEvent) *outboxStoreStub {
	counts := map[domain.OutboxStatus]int64{
		domain.OutboxStatusPending: 0,
		domain.OutboxStatusFailed:  0,
		domain.OutboxStatusSent:    0,
	}

	for _, evt := range eventsBatch {
		if evt == nil {
			continue
		}
		counts[evt.Status]++
	}

	return &outboxStoreStub{
		events: eventsBatch,
		counts: counts,
	}
}

func (s *outboxStoreStub) ListOutboxForDispatch(limit int, now time.Time) ([]*domain.OutboxEvent, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	s.lastBatchAt = now
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	return s.events[:limit], nil
}

func (s *outboxStoreStub) MarkOutboxSent(eventID string, sentAt time.Time) error {
	s.sentIDs = append(s.sentIDs, eventID)
	for _, evt := range s.events {
		if evt == nil || evt.ID != eventID {
			continue
		}
		if s.counts[evt.Status] > 0 {
			s.counts[evt.Status]--
		}
		evt.Status = domain.OutboxStatusSent
		evt.SentAt = &sentAt
		evt.LastError = ""
		s.counts[domain.OutboxStatusSent]++
		break
	}
	return nil
}

func (s *outboxStoreStub) MarkOutboxRetry(eventID string, nextRetryAt time.Time, lastError string) error {
	s.retryCalls = append(s.retryCalls, retryCall{
		eventID:     eventID,
		nextRetryAt: nextRetryAt,
		lastError:   lastError,
	})

	for _, evt := range s.events {
		if evt == nil || evt.ID != eventID {
			continue
		}
		if s.counts[evt.Status] > 0 {
			s.counts[evt.Status]--
		}
		evt.Status = domain.OutboxStatusFailed
		evt.Attempts++
		evt.NextRetryAt = nextRetryAt
		evt.LastError = lastError
		s.counts[domain.OutboxStatusFailed]++
		break
	}

	return nil
}

func (s *outboxStoreStub) CountOutboxByStatus() (map[domain.OutboxStatus]int64, error) {
	if s.countErr != nil {
		return nil, s.countErr
	}
	s.countCalls++

	return map[domain.OutboxStatus]int64{
		domain.OutboxStatusPending: s.counts[domain.OutboxStatusPending],
		domain.OutboxStatusFailed:  s.counts[domain.OutboxStatusFailed],
		domain.OutboxStatusSent:    s.counts[domain.OutboxStatusSent],
	}, nil
}

func runDispatcherOnce(dispatcher *events.OutboxDispatcher) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	dispatcher.Run(ctx)
}

func TestOutboxDispatcherRun_SendsAndMarksSent(t *testing.T) {
	store := newOutboxStoreStub(&domain.OutboxEvent{
		ID:          "evt-1",
		AggregateID: "question-1",
		EventType:   events.EventQuestionCreated,
		Payload:     `{"event_id":"evt-1"}`,
		Status:      domain.OutboxStatusPending,
	})
	publisher := &outboxPublisherStub{}
	dispatcher := events.NewOutboxDispatcher(store, publisher, 10, time.Hour)

	runDispatcherOnce(dispatcher)

	if len(publisher.calls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(publisher.calls))
	}
	if publisher.calls[0].eventType != events.EventQuestionCreated {
		t.Fatalf("expected event type %q, got %q", events.EventQuestionCreated, publisher.calls[0].eventType)
	}
	if publisher.calls[0].key != "question-1" {
		t.Fatalf("expected key question-1, got %q", publisher.calls[0].key)
	}
	if len(store.sentIDs) != 1 || store.sentIDs[0] != "evt-1" {
		t.Fatalf("expected sent id evt-1, got %#v", store.sentIDs)
	}
	if len(store.retryCalls) != 0 {
		t.Fatalf("expected no retry calls, got %#v", store.retryCalls)
	}
	if store.countCalls == 0 {
		t.Fatalf("expected queue size refresh call")
	}
}

func TestOutboxDispatcherRun_RetryOnPublishFailure(t *testing.T) {
	startedAt := time.Now().UTC()
	store := newOutboxStoreStub(&domain.OutboxEvent{
		ID:        "evt-2",
		EventType: events.EventQuestionUpdated,
		Payload:   `{"event_id":"evt-2"}`,
		Status:    domain.OutboxStatusPending,
		Attempts:  1,
	})
	publisher := &outboxPublisherStub{err: errors.New("kafka unavailable")}
	dispatcher := events.NewOutboxDispatcher(store, publisher, 10, time.Hour)

	runDispatcherOnce(dispatcher)

	if len(publisher.calls) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(publisher.calls))
	}
	if publisher.calls[0].key != "evt-2" {
		t.Fatalf("expected fallback key evt-2, got %q", publisher.calls[0].key)
	}
	if len(store.sentIDs) != 0 {
		t.Fatalf("expected no sent ids, got %#v", store.sentIDs)
	}
	if len(store.retryCalls) != 1 {
		t.Fatalf("expected 1 retry call, got %#v", store.retryCalls)
	}
	if store.retryCalls[0].eventID != "evt-2" {
		t.Fatalf("expected retry event id evt-2, got %q", store.retryCalls[0].eventID)
	}
	if store.retryCalls[0].lastError != "kafka unavailable" {
		t.Fatalf("expected retry reason kafka unavailable, got %q", store.retryCalls[0].lastError)
	}
	if delay := store.retryCalls[0].nextRetryAt.Sub(startedAt); delay < 2*time.Second || delay > 4*time.Second {
		t.Fatalf("expected retry delay around 2s, got %v", delay)
	}
}

func TestOutboxDispatcherRun_RetryDelayCappedAtThirtySeconds(t *testing.T) {
	startedAt := time.Now().UTC()
	store := newOutboxStoreStub(&domain.OutboxEvent{
		ID:        "evt-3",
		EventType: events.EventQuestionUpdated,
		Payload:   `{"event_id":"evt-3"}`,
		Status:    domain.OutboxStatusPending,
		Attempts:  100,
	})
	publisher := &outboxPublisherStub{err: errors.New("kafka unavailable")}
	dispatcher := events.NewOutboxDispatcher(store, publisher, 10, time.Hour)

	runDispatcherOnce(dispatcher)

	if len(store.retryCalls) != 1 {
		t.Fatalf("expected 1 retry call, got %#v", store.retryCalls)
	}
	if delay := store.retryCalls[0].nextRetryAt.Sub(startedAt); delay < 29*time.Second || delay > 31*time.Second {
		t.Fatalf("expected retry delay around 30s, got %v", delay)
	}
}

func TestOutboxDispatcherRun_RetryOnEmptyPayload(t *testing.T) {
	startedAt := time.Now().UTC()
	store := newOutboxStoreStub(&domain.OutboxEvent{
		ID:        "evt-4",
		EventType: events.EventQuestionDeleted,
		Payload:   "   ",
		Status:    domain.OutboxStatusPending,
		Attempts:  0,
	})
	publisher := &outboxPublisherStub{}
	dispatcher := events.NewOutboxDispatcher(store, publisher, 10, time.Hour)

	runDispatcherOnce(dispatcher)

	if len(publisher.calls) != 0 {
		t.Fatalf("expected no publish call for empty payload, got %d", len(publisher.calls))
	}
	if len(store.retryCalls) != 1 {
		t.Fatalf("expected 1 retry call, got %#v", store.retryCalls)
	}
	if store.retryCalls[0].lastError != "empty outbox payload" {
		t.Fatalf("expected empty payload error, got %q", store.retryCalls[0].lastError)
	}
	if delay := store.retryCalls[0].nextRetryAt.Sub(startedAt); delay < 1*time.Second || delay > 3*time.Second {
		t.Fatalf("expected retry delay around 1s, got %v", delay)
	}
}

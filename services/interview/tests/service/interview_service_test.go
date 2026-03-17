package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"easyoffer/interview/internal/domain"
	"easyoffer/interview/internal/repository"
	interviewservice "easyoffer/interview/internal/service"
)

type sessionRepositoryStub struct {
	session   *domain.InterviewSession
	getErr    error
	saveErr   error
	saveCalls int
	getCalls  int
}

func (s *sessionRepositoryStub) Save(_ context.Context, session *domain.InterviewSession) error {
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.session = session
	return nil
}

func (s *sessionRepositoryStub) Get(_ context.Context, _ string) (*domain.InterviewSession, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.session == nil {
		return nil, repository.ErrSessionNotFound
	}
	return s.session, nil
}

func (s *sessionRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

type eventStoreStub struct {
	appendErr error
	listErr   error
	listCalls int
	events    []domain.InterviewEvent
}

func (s *eventStoreStub) Append(_ context.Context, event *domain.InterviewEvent) error {
	if s.appendErr != nil {
		return s.appendErr
	}
	if event != nil {
		s.events = append(s.events, *event)
	}
	return nil
}

func (s *eventStoreStub) ListBySession(_ context.Context, _ string) ([]domain.InterviewEvent, error) {
	s.listCalls++
	if s.listErr != nil {
		return nil, s.listErr
	}
	result := make([]domain.InterviewEvent, 0, len(s.events))
	for i := range s.events {
		result = append(result, s.events[i])
	}
	return result, nil
}

func mustRawJSON(t *testing.T, payload interface{}) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return raw
}

type questionRepositoryStub struct {
	listResult []domain.QuestionSnapshot
	listErr    error
}

func (q *questionRepositoryStub) Upsert(_ context.Context, _ *domain.QuestionSnapshot) error {
	return nil
}

func (q *questionRepositoryStub) DeleteQuestion(_ context.Context, _ string) error {
	return nil
}

func (q *questionRepositoryStub) List(_ context.Context, _ repository.QuestionFilter) ([]domain.QuestionSnapshot, error) {
	return q.listResult, q.listErr
}

func TestGetNextQuestion_WhenAllQuestionsAnswered_ReturnsDoneStateWithoutError(t *testing.T) {
	now := time.Now().UTC()
	repo := &sessionRepositoryStub{
		session: &domain.InterviewSession{
			ID:           "session-1",
			UserID:       "user-1",
			Questions:    []domain.QuestionSnapshot{{ID: "q-1", Title: "Q1"}},
			CurrentIndex: 0,
			Answers: map[string]domain.SessionAnswer{
				"q-1": {QuestionID: "q-1", Status: domain.StatusKnow, AnsweredAt: now},
			},
			StartedAt: now,
		},
	}

	svc := interviewservice.NewInterviewService(repo, &questionRepositoryStub{}, time.Minute)

	question, hasMore, err := svc.GetNextQuestion(context.Background(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if question != nil {
		t.Fatalf("expected nil question when interview is done")
	}
	if hasMore {
		t.Fatalf("expected hasMore=false when interview is done")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no repository save when interview is done")
	}
}

func TestSubmitAnswer_InvalidStatus_ReturnsErrInvalidStatus(t *testing.T) {
	repo := &sessionRepositoryStub{}
	svc := interviewservice.NewInterviewService(repo, &questionRepositoryStub{}, time.Minute)

	err := svc.SubmitAnswer(context.Background(), "user-1", "session-1", interviewservice.SubmitAnswerInput{
		QuestionID: "q-1",
		Status:     "invalid-status",
	})

	if !errors.Is(err, interviewservice.ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got: %v", err)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no repository save for invalid status")
	}
	if repo.getCalls != 0 {
		t.Fatalf("expected no repository get for invalid status")
	}
}

func TestGetResult_BeforeFinish_ReturnsErrSessionNotFinished(t *testing.T) {
	now := time.Now().UTC()
	repo := &sessionRepositoryStub{
		session: &domain.InterviewSession{
			ID:        "session-1",
			UserID:    "user-1",
			Questions: []domain.QuestionSnapshot{{ID: "q-1"}},
			Answers:   map[string]domain.SessionAnswer{},
			StartedAt: now,
		},
	}

	svc := interviewservice.NewInterviewService(repo, &questionRepositoryStub{}, time.Minute)

	result, err := svc.GetResult(context.Background(), "user-1", "session-1")
	if !errors.Is(err, interviewservice.ErrSessionNotFinished) {
		t.Fatalf("expected ErrSessionNotFinished, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result before finish")
	}
}

func TestStartSession_AppendsSessionStartedEvent(t *testing.T) {
	repo := &sessionRepositoryStub{}
	questions := &questionRepositoryStub{
		listResult: []domain.QuestionSnapshot{{ID: "q-1", Title: "Question 1", Category: "theory", AnswerFormat: "text"}},
	}
	eventStore := &eventStoreStub{}
	svc := interviewservice.NewInterviewServiceWithEventStore(repo, questions, eventStore, time.Minute)

	session, firstQuestion, err := svc.StartSession(context.Background(), "user-1", interviewservice.StartSessionInput{
		Category:     "theory",
		AnswerFormat: "text",
		Language:     "go",
		Count:        1,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if session == nil || firstQuestion == nil {
		t.Fatalf("expected non-nil session and first question")
	}
	if len(eventStore.events) != 1 {
		t.Fatalf("expected 1 appended event, got %d", len(eventStore.events))
	}

	event := eventStore.events[0]
	if event.Type != domain.EventSessionStarted {
		t.Fatalf("expected event type %q, got %q", domain.EventSessionStarted, event.Type)
	}
	if event.SessionID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, event.SessionID)
	}

	var payload domain.SessionStartedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.RequestedCount != 1 {
		t.Fatalf("expected requested_count=1, got %d", payload.RequestedCount)
	}
	if len(payload.Questions) != 1 || payload.Questions[0].ID != "q-1" {
		t.Fatalf("unexpected session started questions payload: %#v", payload.Questions)
	}
}

func TestSubmitAnswer_AppendsAnswerSubmittedEvent(t *testing.T) {
	now := time.Now().UTC()
	repo := &sessionRepositoryStub{
		session: &domain.InterviewSession{
			ID:           "session-1",
			UserID:       "user-1",
			Questions:    []domain.QuestionSnapshot{{ID: "q-1"}},
			CurrentIndex: 0,
			Answers:      map[string]domain.SessionAnswer{},
			StartedAt:    now,
		},
	}
	eventStore := &eventStoreStub{}
	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	err := svc.SubmitAnswer(context.Background(), "user-1", "session-1", interviewservice.SubmitAnswerInput{
		QuestionID: "q-1",
		Status:     string(domain.StatusKnow),
		UserAnswer: "answer",
		Note:       "note",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(eventStore.events) != 1 {
		t.Fatalf("expected 1 appended event, got %d", len(eventStore.events))
	}

	event := eventStore.events[0]
	if event.Type != domain.EventAnswerSubmitted {
		t.Fatalf("expected event type %q, got %q", domain.EventAnswerSubmitted, event.Type)
	}

	var payload domain.AnswerSubmittedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.QuestionID != "q-1" {
		t.Fatalf("expected question id q-1, got %q", payload.QuestionID)
	}
	if payload.Status != domain.StatusKnow {
		t.Fatalf("expected status know, got %q", payload.Status)
	}
}

func TestFinishSession_AppendsSessionFinishedEvent(t *testing.T) {
	now := time.Now().UTC()
	repo := &sessionRepositoryStub{
		session: &domain.InterviewSession{
			ID:        "session-1",
			UserID:    "user-1",
			Questions: []domain.QuestionSnapshot{{ID: "q-1"}},
			Answers: map[string]domain.SessionAnswer{
				"q-1": {QuestionID: "q-1", Status: domain.StatusKnow, AnsweredAt: now},
			},
			StartedAt: now,
		},
	}
	eventStore := &eventStoreStub{}
	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	result, err := svc.FinishSession(context.Background(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
	if len(eventStore.events) != 1 {
		t.Fatalf("expected 1 appended event, got %d", len(eventStore.events))
	}

	event := eventStore.events[0]
	if event.Type != domain.EventSessionFinished {
		t.Fatalf("expected event type %q, got %q", domain.EventSessionFinished, event.Type)
	}

	var payload domain.SessionFinishedPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload.FinishedAt.IsZero() {
		t.Fatalf("expected non-zero finished_at in payload")
	}
}

func TestGetResult_RebuildsFromEventStoreWhenSnapshotMissing(t *testing.T) {
	startedAt := time.Now().UTC().Add(-2 * time.Minute)
	answeredAt := startedAt.Add(30 * time.Second)
	finishedAt := startedAt.Add(1 * time.Minute)

	repo := &sessionRepositoryStub{}
	eventStore := &eventStoreStub{
		events: []domain.InterviewEvent{
			{
				ID:         "evt-1",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventSessionStarted,
				OccurredAt: startedAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.SessionStartedPayload{
					Questions: []domain.QuestionSnapshot{{ID: "q-1"}, {ID: "q-2"}},
					StartedAt: startedAt,
				}),
			},
			{
				ID:         "evt-2",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventAnswerSubmitted,
				OccurredAt: answeredAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.AnswerSubmittedPayload{
					QuestionID: "q-1",
					Status:     domain.StatusKnow,
					AnsweredAt: answeredAt,
				}),
			},
			{
				ID:         "evt-3",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventSessionFinished,
				OccurredAt: finishedAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.SessionFinishedPayload{
					FinishedAt: finishedAt,
				}),
			},
		},
	}

	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	result, err := svc.GetResult(context.Background(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
	if result.Total != 2 {
		t.Fatalf("expected total=2, got %d", result.Total)
	}
	if result.Answered != 1 || result.Know != 1 {
		t.Fatalf("unexpected counters: answered=%d know=%d", result.Answered, result.Know)
	}
	if eventStore.listCalls == 0 {
		t.Fatalf("expected event store to be used for replay")
	}
}

func TestGetNextQuestion_RebuildsFromEventStoreWhenSnapshotMissing(t *testing.T) {
	startedAt := time.Now().UTC().Add(-2 * time.Minute)
	answeredAt := startedAt.Add(30 * time.Second)

	repo := &sessionRepositoryStub{}
	eventStore := &eventStoreStub{
		events: []domain.InterviewEvent{
			{
				ID:         "evt-1",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventSessionStarted,
				OccurredAt: startedAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.SessionStartedPayload{
					Questions: []domain.QuestionSnapshot{{ID: "q-1"}, {ID: "q-2"}},
					StartedAt: startedAt,
				}),
			},
			{
				ID:         "evt-2",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventAnswerSubmitted,
				OccurredAt: answeredAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.AnswerSubmittedPayload{
					QuestionID: "q-1",
					Status:     domain.StatusKnow,
					AnsweredAt: answeredAt,
				}),
			},
		},
	}

	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	question, hasMore, err := svc.GetNextQuestion(context.Background(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true")
	}
	if question == nil || question.ID != "q-2" {
		t.Fatalf("expected next question q-2, got %#v", question)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected read-only query path without snapshot writes, got save calls=%d", repo.saveCalls)
	}
}

func TestGetResult_NoSnapshotAndNoEvents_ReturnsErrSessionNotFound(t *testing.T) {
	repo := &sessionRepositoryStub{}
	eventStore := &eventStoreStub{events: []domain.InterviewEvent{}}
	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	result, err := svc.GetResult(context.Background(), "user-1", "missing-session")
	if !errors.Is(err, interviewservice.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result")
	}
}

func TestReplaySession_RebuildsAndPersistsSnapshot(t *testing.T) {
	startedAt := time.Now().UTC().Add(-1 * time.Minute)
	answeredAt := startedAt.Add(20 * time.Second)

	repo := &sessionRepositoryStub{}
	eventStore := &eventStoreStub{
		events: []domain.InterviewEvent{
			{
				ID:         "evt-1",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventSessionStarted,
				OccurredAt: startedAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.SessionStartedPayload{
					Questions: []domain.QuestionSnapshot{{ID: "q-1"}, {ID: "q-2"}},
					StartedAt: startedAt,
				}),
			},
			{
				ID:         "evt-2",
				SessionID:  "session-1",
				UserID:     "user-1",
				Type:       domain.EventAnswerSubmitted,
				OccurredAt: answeredAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.AnswerSubmittedPayload{
					QuestionID: "q-1",
					Status:     domain.StatusKnow,
					AnsweredAt: answeredAt,
				}),
			},
		},
	}

	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	session, err := svc.ReplaySession(context.Background(), "user-1", "session-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if session == nil {
		t.Fatalf("expected non-nil session")
	}
	if session.ID != "session-1" {
		t.Fatalf("expected session id=session-1, got %q", session.ID)
	}
	if len(session.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(session.Questions))
	}
	if len(session.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(session.Answers))
	}
	if repo.saveCalls != 1 {
		t.Fatalf("expected snapshot save after replay, got %d", repo.saveCalls)
	}
	if repo.session == nil {
		t.Fatalf("expected repository to persist replayed session")
	}
}

func TestReplaySession_WhenUserMismatch_ReturnsErrSessionForbidden(t *testing.T) {
	startedAt := time.Now().UTC().Add(-1 * time.Minute)

	repo := &sessionRepositoryStub{}
	eventStore := &eventStoreStub{
		events: []domain.InterviewEvent{
			{
				ID:         "evt-1",
				SessionID:  "session-1",
				UserID:     "another-user",
				Type:       domain.EventSessionStarted,
				OccurredAt: startedAt,
				Version:    1,
				Payload: mustRawJSON(t, domain.SessionStartedPayload{
					Questions: []domain.QuestionSnapshot{{ID: "q-1"}},
					StartedAt: startedAt,
				}),
			},
		},
	}

	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	session, err := svc.ReplaySession(context.Background(), "user-1", "session-1")
	if !errors.Is(err, interviewservice.ErrSessionForbidden) {
		t.Fatalf("expected ErrSessionForbidden, got: %v", err)
	}
	if session != nil {
		t.Fatalf("expected nil session")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no snapshot save when forbidden")
	}
}

func TestReplaySession_NoEvents_ReturnsErrSessionNotFound(t *testing.T) {
	repo := &sessionRepositoryStub{}
	eventStore := &eventStoreStub{events: []domain.InterviewEvent{}}
	svc := interviewservice.NewInterviewServiceWithEventStore(repo, &questionRepositoryStub{}, eventStore, time.Minute)

	session, err := svc.ReplaySession(context.Background(), "user-1", "missing-session")
	if !errors.Is(err, interviewservice.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
	if session != nil {
		t.Fatalf("expected nil session")
	}
	if repo.saveCalls != 0 {
		t.Fatalf("expected no snapshot save when events are missing")
	}
}

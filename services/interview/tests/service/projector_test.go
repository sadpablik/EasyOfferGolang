package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"easyoffer/interview/internal/domain"
	interviewservice "easyoffer/interview/internal/service"
)

type projectorEventStoreStub struct {
	sessionIDs []string
	counts     map[string]int64
	streams    map[string][]domain.InterviewEvent
	listErr    error
	countErr   error
	streamErr  error
}

func (s *projectorEventStoreStub) Append(_ context.Context, _ *domain.InterviewEvent) error {
	return nil
}

func (s *projectorEventStoreStub) ListBySession(_ context.Context, sessionID string) ([]domain.InterviewEvent, error) {
	if s.streamErr != nil {
		return nil, s.streamErr
	}
	items := s.streams[sessionID]
	result := make([]domain.InterviewEvent, 0, len(items))
	for i := range items {
		result = append(result, items[i])
	}
	return result, nil
}

func (s *projectorEventStoreStub) ListSessions(_ context.Context) ([]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	result := make([]string, 0, len(s.sessionIDs))
	result = append(result, s.sessionIDs...)
	return result, nil
}

func (s *projectorEventStoreStub) EventCount(_ context.Context, sessionID string) (int64, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	if value, ok := s.counts[sessionID]; ok {
		return value, nil
	}
	return int64(len(s.streams[sessionID])), nil
}

type projectorSessionRepositoryStub struct {
	sessions  map[string]*domain.InterviewSession
	saveCalls int
	saveErr   error
}

func (s *projectorSessionRepositoryStub) Save(_ context.Context, session *domain.InterviewSession) error {
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	if s.sessions == nil {
		s.sessions = make(map[string]*domain.InterviewSession)
	}
	s.sessions[session.ID] = session
	return nil
}

func (s *projectorSessionRepositoryStub) Get(_ context.Context, sessionID string) (*domain.InterviewSession, error) {
	if value, ok := s.sessions[sessionID]; ok {
		return value, nil
	}
	return nil, errors.New("not found")
}

func (s *projectorSessionRepositoryStub) Delete(_ context.Context, _ string) error {
	return nil
}

type checkpointRepositoryStub struct {
	values   map[string]int64
	setCalls int
	getErr   error
	setErr   error
}

func (s *checkpointRepositoryStub) Get(_ context.Context, sessionID string) (int64, error) {
	if s.getErr != nil {
		return 0, s.getErr
	}
	if s.values == nil {
		return 0, nil
	}
	return s.values[sessionID], nil
}

func (s *checkpointRepositoryStub) Set(_ context.Context, sessionID string, eventCount int64) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.setCalls++
	if s.values == nil {
		s.values = make(map[string]int64)
	}
	s.values[sessionID] = eventCount
	return nil
}

func mustProjectorPayload(t *testing.T, payload interface{}) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return raw
}

func TestProjectorProjectOnceProjectsNewEventsAndUpdatesCheckpoint(t *testing.T) {
	startedAt := time.Now().UTC().Add(-1 * time.Minute)
	answeredAt := startedAt.Add(20 * time.Second)

	events := &projectorEventStoreStub{
		sessionIDs: []string{"session-1"},
		counts:     map[string]int64{"session-1": 2},
		streams: map[string][]domain.InterviewEvent{
			"session-1": {
				{
					ID:         "evt-1",
					SessionID:  "session-1",
					UserID:     "user-1",
					Type:       domain.EventSessionStarted,
					OccurredAt: startedAt,
					Payload: mustProjectorPayload(t, domain.SessionStartedPayload{
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
					Payload: mustProjectorPayload(t, domain.AnswerSubmittedPayload{
						QuestionID: "q-1",
						Status:     domain.StatusKnow,
						AnsweredAt: answeredAt,
					}),
				},
			},
		},
	}
	sessions := &projectorSessionRepositoryStub{}
	checkpoints := &checkpointRepositoryStub{}

	projector := interviewservice.NewInterviewProjector(events, sessions, checkpoints, time.Second)
	if err := projector.ProjectOnce(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if sessions.saveCalls != 1 {
		t.Fatalf("expected one projected save, got %d", sessions.saveCalls)
	}
	projected := sessions.sessions["session-1"]
	if projected == nil {
		t.Fatalf("expected projected session to be persisted")
	}
	if len(projected.Answers) != 1 {
		t.Fatalf("expected one projected answer, got %d", len(projected.Answers))
	}
	if checkpoints.values["session-1"] != 2 {
		t.Fatalf("expected checkpoint=2, got %d", checkpoints.values["session-1"])
	}
}

func TestProjectorProjectOnceSkipsUpToDateCheckpoint(t *testing.T) {
	events := &projectorEventStoreStub{
		sessionIDs: []string{"session-1"},
		counts:     map[string]int64{"session-1": 2},
		streams: map[string][]domain.InterviewEvent{
			"session-1": {{Type: domain.EventSessionStarted}},
		},
	}
	sessions := &projectorSessionRepositoryStub{}
	checkpoints := &checkpointRepositoryStub{values: map[string]int64{"session-1": 2}}

	projector := interviewservice.NewInterviewProjector(events, sessions, checkpoints, time.Second)
	if err := projector.ProjectOnce(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if sessions.saveCalls != 0 {
		t.Fatalf("expected no projected save when checkpoint is up to date, got %d", sessions.saveCalls)
	}
	if checkpoints.setCalls != 0 {
		t.Fatalf("expected no checkpoint update when up to date, got %d", checkpoints.setCalls)
	}
}

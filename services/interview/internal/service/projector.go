package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"easyoffer/interview/internal/repository"

	"github.com/prometheus/client_golang/prometheus"
)

var projectorRunsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "interview",
		Name:      "projector_runs_total",
		Help:      "Total number of projector runs by status.",
	},
	[]string{"status"},
)

var projectorProjectedSessionsTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "easyoffer",
		Subsystem: "interview",
		Name:      "projector_projected_sessions_total",
		Help:      "Total number of session projections applied by projector.",
	},
)

var projectorMaxLagEvents = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "easyoffer",
		Subsystem: "interview",
		Name:      "projector_max_lag_events",
		Help:      "Maximum lag in events between stream length and checkpoint across sessions.",
	},
)

var registerProjectorMetricsOnce sync.Once

func RegisterProjectorMetrics(reg prometheus.Registerer) {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	registerProjectorMetricsOnce.Do(func() {
		reg.MustRegister(projectorRunsTotal, projectorProjectedSessionsTotal, projectorMaxLagEvents)
	})
}

type InterviewProjector struct {
	events       repository.ProjectorEventStore
	sessions     repository.SessionRepository
	checkpoints  repository.ProjectionCheckpointRepository
	pollInterval time.Duration
}

func NewInterviewProjector(
	events repository.ProjectorEventStore,
	sessions repository.SessionRepository,
	checkpoints repository.ProjectionCheckpointRepository,
	pollInterval time.Duration,
) *InterviewProjector {
	if pollInterval <= 0 {
		pollInterval = 1 * time.Second
	}

	return &InterviewProjector{
		events:       events,
		sessions:     sessions,
		checkpoints:  checkpoints,
		pollInterval: pollInterval,
	}
}

func (p *InterviewProjector) Run(ctx context.Context) error {
	if p == nil || p.events == nil || p.sessions == nil {
		return nil
	}

	if err := p.ProjectOnce(ctx); err != nil {
		projectorRunsTotal.WithLabelValues("error").Inc()
		log.Printf("interview projector initial sync failed: %v", err)
	} else {
		projectorRunsTotal.WithLabelValues("success").Inc()
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.ProjectOnce(ctx); err != nil {
				projectorRunsTotal.WithLabelValues("error").Inc()
				log.Printf("interview projector sync failed: %v", err)
			} else {
				projectorRunsTotal.WithLabelValues("success").Inc()
			}
		}
	}
}

func (p *InterviewProjector) ProjectOnce(ctx context.Context) error {
	if p == nil || p.events == nil || p.sessions == nil {
		return nil
	}

	sessionIDs, err := p.events.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list event sessions: %w", err)
	}

	var projectedSessions int
	var maxLag int64

	for _, sessionID := range sessionIDs {
		eventCount, err := p.events.EventCount(ctx, sessionID)
		if err != nil {
			log.Printf("failed to get event count for session %s: %v", sessionID, err)
			continue
		}
		if eventCount == 0 {
			continue
		}

		checkpoint, err := p.checkpoints.Get(ctx, sessionID)
		if err != nil {
			log.Printf("failed to get projection checkpoint for session %s: %v", sessionID, err)
			continue
		}
		lag := eventCount - checkpoint
		if lag > maxLag {
			maxLag = lag
		}
		if checkpoint >= eventCount {
			continue
		}

		if err := p.projectSessionWithRetry(ctx, sessionID); err != nil {
			log.Printf("failed to project session %s after retries: %v", sessionID, err)
			continue
		}

		projectedSessions++
	}

	projectorProjectedSessionsTotal.Add(float64(projectedSessions))
	projectorMaxLagEvents.Set(float64(maxLag))

	return nil
}

func (p *InterviewProjector) projectSessionWithRetry(ctx context.Context, sessionID string) error {
	const maxRetries = 3
	var backoff = 1 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := p.projectSession(ctx, sessionID)
		if err == nil {
			return nil
		}

		if attempt < maxRetries-1 {
			select {
			case <-time.After(backoff):
				backoff *= 2
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("projection failed after %d retries", maxRetries)
}

func (p *InterviewProjector) projectSession(ctx context.Context, sessionID string) error {
	events, err := p.events.ListBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session stream: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	session, err := replaySessionFromEvents(events)
	if err != nil {
		return fmt.Errorf("failed to replay stream: %w", err)
	}

	if err := p.sessions.Save(ctx, session); err != nil {
		return fmt.Errorf("failed to persist projected session: %w", err)
	}

	if err := p.checkpoints.Set(ctx, sessionID, int64(len(events))); err != nil {
		return fmt.Errorf("failed to save projection checkpoint: %w", err)
	}

	return nil
}

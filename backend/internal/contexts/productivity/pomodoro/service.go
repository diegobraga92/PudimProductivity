package pomodoro

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/pkg/uuid"
)

const timerTickInterval = 1 * time.Second

type NoiseProvider interface {
	Play(ctx context.Context, trackID string) error
	Stop(ctx context.Context) error
}

type PomodoroService struct {
	mu      sync.Mutex
	current *PomodoroSession
	noise   NoiseProvider // optional, may be nil
	audit   audit.Logger
	bus     eventbus.Bus       // Phase 9a: publishes session lifecycle events
	cancel  context.CancelFunc // cancels the current timer goroutine
}

func NewPomodoroService(noise NoiseProvider, auditLogger audit.Logger, bus eventbus.Bus) *PomodoroService {
	if auditLogger == nil {
		auditLogger = audit.NoopLogger{}
	}
	return &PomodoroService{
		noise: noise,
		audit: auditLogger,
		bus:   bus,
	}
}

// publish emits a session lifecycle event. A nil bus is a no-op; failures are
// logged, never propagated (the session lifecycle is the source of truth).
func (s *PomodoroService) publish(ctx context.Context, typ eventbus.EventType, payload any) {
	if s.bus == nil {
		return
	}
	if err := s.bus.Publish(ctx, typ, payload); err != nil {
		log.Warn().Err(err).Str("event_type", string(typ)).Msg("failed to publish pomodoro event")
	}
}

func (s *PomodoroService) StartSession(ctx context.Context, userID string, focusMinutes, breakMinutes int, continuous bool, noise *NoiseConfig) (*PomodoroSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel any existing session and its timer goroutine
	if s.current != nil && s.current.Status != SessionCompleted && s.current.Status != SessionCancelled {
		s.cancelTimer()
		_ = s.current.Cancel()
		if s.noise != nil {
			_ = s.noise.Stop(ctx)
		}
		log.Info().Str("session_id", s.current.ID).Msg("previous session cancelled by new session")
	}

	id := uuid.NewUUID()
	session, err := NewSession(id, focusMinutes, breakMinutes, continuous, noise)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	session.UserID = userID

	s.current = session
	s.startTimer()

	if s.noise != nil && noise != nil && noise.Enabled && noise.TrackID != "" {
		if err := s.noise.Play(ctx, noise.TrackID); err != nil {
			log.Warn().Err(err).Msg("failed to start noise track")
		}
	}

	log.Info().Str("session_id", id).Int("focus_minutes", focusMinutes).Bool("continuous", continuous).Msg("pomodoro session started")

	s.audit.Log(ctx, audit.ActionFocusStarted, audit.ResourcePomodoro, id, nil, map[string]any{
		"focus_minutes": focusMinutes,
		"break_minutes": breakMinutes,
		"continuous":    continuous,
	})

	s.publish(ctx, eventbus.EventPomodoroSessionStarted, map[string]any{
		"session_id":    id,
		"user_id":       userID,
		"phase":         string(PhaseFocus),
		"continuous":    continuous,
		"focus_minutes": focusMinutes,
		"started_at":    session.StartedAt.Format(time.RFC3339),
	})

	return session, nil
}

func (s *PomodoroService) GetCurrent() *PomodoroSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil
	}

	// Return a copy to avoid race conditions on the caller side
	session := *s.current
	return &session
}

func (s *PomodoroService) Pause(ctx context.Context) (*PomodoroSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, fmt.Errorf("no active session")
	}

	if err := s.current.Pause(); err != nil {
		return nil, err
	}

	s.cancelTimer()

	if s.noise != nil {
		_ = s.noise.Stop(ctx)
	}

	log.Info().Str("session_id", s.current.ID).Msg("pomodoro session paused")
	session := *s.current
	return &session, nil
}

func (s *PomodoroService) Resume(ctx context.Context) (*PomodoroSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, fmt.Errorf("no active session")
	}

	if err := s.current.Resume(); err != nil {
		return nil, err
	}

	s.startTimer()

	if s.noise != nil && s.current.NoiseConfig != nil && s.current.NoiseConfig.Enabled {
		_ = s.noise.Play(ctx, s.current.NoiseConfig.TrackID)
	}

	log.Info().Str("session_id", s.current.ID).Msg("pomodoro session resumed")
	session := *s.current
	return &session, nil
}

func (s *PomodoroService) Stop(ctx context.Context) (*PomodoroSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil, fmt.Errorf("no active session")
	}

	s.cancelTimer()

	// Try to complete first; if that fails (e.g. already completed/cancelled), cancel
	if err := s.current.Complete(); err != nil {
		if err := s.current.Cancel(); err != nil {
			return nil, fmt.Errorf("cannot stop session: %w", err)
		}
	}

	if s.noise != nil {
		_ = s.noise.Stop(ctx)
	}

	log.Info().Str("session_id", s.current.ID).Str("status", string(s.current.Status)).Msg("pomodoro session stopped")

	s.audit.Log(ctx, audit.ActionFocusCompleted, audit.ResourcePomodoro, s.current.ID, nil, map[string]any{
		"status":    string(s.current.Status),
		"elapsed_s": int(s.current.Elapsed().Seconds()),
	})

	// Publish completion only for focus segments — breaks are not focus history.
	// A completed break segment that is manually stopped records nothing.
	session := *s.current
	switch {
	case session.Status == SessionCompleted && session.Phase == PhaseFocus:
		s.publish(ctx, eventbus.EventPomodoroSessionCompleted, map[string]any{
			"session_id":    session.ID,
			"user_id":       session.UserID,
			"focus_minutes": int(session.FocusDuration.Minutes()),
			"elapsed_s":     int(session.Elapsed().Seconds()),
			"started_at":    session.StartedAt.Format(time.RFC3339),
			"completed_at":  time.Now().UTC().Format(time.RFC3339),
		})
	case session.Status == SessionCancelled:
		s.publish(ctx, eventbus.EventPomodoroSessionCancelled, map[string]any{
			"session_id": session.ID,
			"user_id":    session.UserID,
		})
	}

	return &session, nil
}

// launches a background goroutine that ticks every second.
// Auto-completes a single-shot session when its focus duration elapses, or
// auto-advances a continuous run to the next phase. Must be called with s.mu held.
func (s *PomodoroService) startTimer() {
	s.cancelTimer() // cancel any previous timer

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	sessionID := s.current.ID

	go func() {
		ticker := time.NewTicker(timerTickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.mu.Lock()
				if s.current != nil && s.current.ID == sessionID && s.current.Status == SessionRunning && s.current.Remaining() <= 0 {
					if !s.current.Continuous {
						// Single-shot: complete the focus session and finish.
						s.completeSegment(ctx)
						s.mu.Unlock()
						return
					}
					// Continuous: advance to the next phase. This creates a new
					// segment and restarts the timer, so this goroutine exits.
					s.advanceSegment(ctx)
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
			}
		}
	}()
}

// completeSegment finalizes a single-shot focus session as completed and
// publishes the completion event consumed by the insights module.
// Must be called with s.mu held.
func (s *PomodoroService) completeSegment(ctx context.Context) {
	sessionID := s.current.ID
	_ = s.current.Complete()
	log.Info().Str("session_id", sessionID).Msg("pomodoro session auto-completed")
	s.audit.Log(context.Background(), audit.ActionFocusCompleted, audit.ResourcePomodoro, sessionID, nil, map[string]any{
		"status":    "completed",
		"auto":      true,
		"elapsed_s": int(s.current.Elapsed().Seconds()),
	})
	s.publish(ctx, eventbus.EventPomodoroSessionCompleted, map[string]any{
		"session_id":    sessionID,
		"user_id":       s.current.UserID,
		"focus_minutes": int(s.current.FocusDuration.Minutes()),
		"elapsed_s":     int(s.current.Elapsed().Seconds()),
		"started_at":    s.current.StartedAt.Format(time.RFC3339),
		"completed_at":  time.Now().UTC().Format(time.RFC3339),
	})
}

// advanceSegment transitions a continuous run from the current segment to the
// next one (focus → break on the same cycle, break → next focus cycle). Every
// segment gets a fresh session id so insights records each focus cycle exactly
// once. A completed focus segment publishes the completion event; a completed
// break segment is silent. Must be called with s.mu held.
func (s *PomodoroService) advanceSegment(ctx context.Context) {
	completed := s.current
	sessionID := completed.ID
	_ = completed.Complete()

	if completed.Phase == PhaseFocus {
		log.Info().Str("session_id", sessionID).Msg("pomodoro focus segment completed, starting break")
		s.audit.Log(context.Background(), audit.ActionFocusCompleted, audit.ResourcePomodoro, sessionID, nil, map[string]any{
			"status":    "completed",
			"auto":      true,
			"phase":     string(PhaseFocus),
			"elapsed_s": int(completed.Elapsed().Seconds()),
		})
		s.publish(ctx, eventbus.EventPomodoroSessionCompleted, map[string]any{
			"session_id":    sessionID,
			"user_id":       completed.UserID,
			"focus_minutes": int(completed.FocusDuration.Minutes()),
			"elapsed_s":     int(completed.Elapsed().Seconds()),
			"started_at":    completed.StartedAt.Format(time.RFC3339),
			"completed_at":  time.Now().UTC().Format(time.RFC3339),
		})
	} else {
		log.Info().Str("session_id", sessionID).Int("cycle", completed.CurrentCycle).Msg("pomodoro break segment completed, starting next focus")
		s.audit.Log(context.Background(), audit.ActionFocusStarted, audit.ResourcePomodoro, completed.ID, nil, map[string]any{
			"phase": string(PhaseBreak),
			"cycle": completed.CurrentCycle,
		})
	}

	next, err := completed.NextSegment(uuid.NewUUID(), time.Now().UTC())
	if err != nil {
		log.Error().Err(err).Msg("failed to create next pomodoro segment")
		s.cancelTimer()
		return
	}

	s.current = next
	s.startTimer()
}

// cancelTimer cancels the current timer goroutine if one is running.
// Must be called with s.mu held.
func (s *PomodoroService) cancelTimer() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

package pomodoro

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/audit"
	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
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

func (s *PomodoroService) StartSession(ctx context.Context, userID string, focusMinutes, breakMinutes int, noise *NoiseConfig) (*PomodoroSession, error) {
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

	id := shared.NewUUID()
	session, err := NewSession(id, focusMinutes, breakMinutes, noise)
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

	log.Info().Str("session_id", id).Int("focus_minutes", focusMinutes).Msg("pomodoro session started")

	s.audit.Log(ctx, audit.ActionFocusStarted, audit.ResourcePomodoro, id, nil, map[string]any{
		"focus_minutes": focusMinutes,
		"break_minutes": breakMinutes,
	})

	s.publish(ctx, eventbus.EventPomodoroSessionStarted, map[string]any{
		"session_id":    id,
		"user_id":       userID,
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

	// Phase 9a: publish the lifecycle event so consumers (e.g. insights) can
	// record focus history.
	session := *s.current
	if session.Status == SessionCompleted {
		s.publish(ctx, eventbus.EventPomodoroSessionCompleted, map[string]any{
			"session_id":    session.ID,
			"user_id":       session.UserID,
			"focus_minutes": int(session.FocusDuration.Minutes()),
			"elapsed_s":     int(session.Elapsed().Seconds()),
			"started_at":    session.StartedAt.Format(time.RFC3339),
			"completed_at":  time.Now().UTC().Format(time.RFC3339),
		})
	} else {
		s.publish(ctx, eventbus.EventPomodoroSessionCancelled, map[string]any{
			"session_id": session.ID,
			"user_id":    session.UserID,
		})
	}

	return &session, nil
}

// launches a background goroutine that ticks every second.
// Auto-completes the session when the focus duration elapses, must be called with s.mu held
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
				if s.current != nil && s.current.ID == sessionID && s.current.Status == SessionRunning {
					if s.current.Remaining() <= 0 {
						_ = s.current.Complete()
						log.Info().Str("session_id", sessionID).Msg("pomodoro session auto-completed")
						s.audit.Log(context.Background(), audit.ActionFocusCompleted, audit.ResourcePomodoro, sessionID, nil, map[string]any{
							"status":    "completed",
							"auto":      true,
							"elapsed_s": int(s.current.Elapsed().Seconds()),
						})
						// Phase 9a: notify consumers of the completed session.
						s.publish(context.Background(), eventbus.EventPomodoroSessionCompleted, map[string]any{
							"session_id":    sessionID,
							"user_id":       s.current.UserID,
							"focus_minutes": int(s.current.FocusDuration.Minutes()),
							"elapsed_s":     int(s.current.Elapsed().Seconds()),
							"started_at":    s.current.StartedAt.Format(time.RFC3339),
							"completed_at":  time.Now().UTC().Format(time.RFC3339),
						})
						s.mu.Unlock()
						return
					}
				}
				s.mu.Unlock()
			}
		}
	}()
}

// cancelTimer cancels the current timer goroutine if one is running.
// Must be called with s.mu held.
func (s *PomodoroService) cancelTimer() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

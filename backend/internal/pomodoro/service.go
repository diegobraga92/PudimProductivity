package pomodoro

import (
	"context"
	"fmt"
	"sync"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
	"github.com/rs/zerolog/log"
)

// NoiseProvider is a provision for future white noise integration.
// When white noise is added later, implement this interface and inject it
// into PomodoroService. No changes to the pomodoro domain are needed.
type NoiseProvider interface {
	Play(ctx context.Context, trackID string) error
	Stop(ctx context.Context) error
}

type PomodoroService struct {
	mu      sync.Mutex
	current *PomodoroSession
	noise   NoiseProvider // optional, may be nil
}

func NewPomodoroService(noise NoiseProvider) *PomodoroService {
	return &PomodoroService{
		noise: noise,
	}
}

func (s *PomodoroService) StartSession(ctx context.Context, focusMinutes, breakMinutes int, noise *NoiseConfig) (*PomodoroSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel any existing session
	if s.current != nil && s.current.Status != SessionCompleted && s.current.Status != SessionCancelled {
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

	s.current = session

	if s.noise != nil && noise != nil && noise.Enabled && noise.TrackID != "" {
		if err := s.noise.Play(ctx, noise.TrackID); err != nil {
			log.Warn().Err(err).Msg("failed to start noise track")
		}
	}

	log.Info().Str("session_id", id).Int("focus_minutes", focusMinutes).Msg("pomodoro session started")
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
	session := *s.current
	return &session, nil
}

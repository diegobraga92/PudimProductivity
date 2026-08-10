package pomodoro

import (
	"fmt"
	"time"
)

type SessionStatus string

const (
	SessionRunning   SessionStatus = "running"
	SessionPaused    SessionStatus = "paused"
	SessionCompleted SessionStatus = "completed"
	SessionCancelled SessionStatus = "cancelled"
)

func (s SessionStatus) Valid() bool {
	switch s {
	case SessionRunning, SessionPaused, SessionCompleted, SessionCancelled:
		return true
	default:
		return false
	}
}

type NoiseConfig struct {
	Enabled bool   `json:"enabled"`
	TrackID string `json:"track_id,omitempty"`
}

type PomodoroSession struct {
	ID                 string
	UserID             string // Phase 9a: who owns the session (for insights)
	Status             SessionStatus
	FocusDuration      time.Duration
	BreakDuration      time.Duration
	CurrentCycle       int           // which pomodoro cycle, for long break tracking
	AccumulatedElapsed time.Duration // holds elapsed from previous play/pauses
	StartedAt          time.Time     // when the current segment started (session creation or last resume)
	PausedAt           *time.Time    // set on pause (for display), cleared on resume
	CompletedAt        *time.Time
	NoiseConfig        *NoiseConfig
}

func NewSession(id string, focusMinutes, breakMinutes int, noise *NoiseConfig) (*PomodoroSession, error) {
	if id == "" {
		return nil, fmt.Errorf("session id cannot be empty")
	}
	if focusMinutes <= 0 {
		focusMinutes = 25
	}
	if breakMinutes <= 0 {
		breakMinutes = 5
	}

	now := time.Now().UTC()

	return &PomodoroSession{
		ID:                 id,
		Status:             SessionRunning,
		FocusDuration:      time.Duration(focusMinutes) * time.Minute,
		BreakDuration:      time.Duration(breakMinutes) * time.Minute,
		CurrentCycle:       1,
		AccumulatedElapsed: 0,
		StartedAt:          now,
		NoiseConfig:        noise,
	}, nil
}

func (s *PomodoroSession) Elapsed() time.Duration {
	if s.Status == SessionRunning {
		return s.AccumulatedElapsed + time.Since(s.StartedAt)
	}
	return s.AccumulatedElapsed
}

func (s *PomodoroSession) Remaining() time.Duration {
	remaining := s.FocusDuration - s.Elapsed()
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *PomodoroSession) Pause() error {
	if s.Status != SessionRunning {
		return fmt.Errorf("cannot pause a session that is %s", s.Status)
	}
	now := time.Now().UTC()

	s.AccumulatedElapsed += now.Sub(s.StartedAt)
	s.PausedAt = &now
	s.Status = SessionPaused
	return nil
}

func (s *PomodoroSession) Resume() error {
	if s.Status != SessionPaused {
		return fmt.Errorf("cannot resume a session that is %s", s.Status)
	}
	// Start a new segment; AccumulatedElapsed already holds the elapsed from previous segments.
	s.StartedAt = time.Now().UTC()
	s.PausedAt = nil
	s.Status = SessionRunning
	return nil
}

func (s *PomodoroSession) Complete() error {
	if s.Status != SessionRunning && s.Status != SessionPaused {
		return fmt.Errorf("cannot complete a session that is %s", s.Status)
	}
	now := time.Now().UTC()
	if s.Status == SessionRunning {
		s.AccumulatedElapsed += now.Sub(s.StartedAt)
	}
	s.CompletedAt = &now
	s.Status = SessionCompleted
	return nil
}

func (s *PomodoroSession) Cancel() error {
	if s.Status == SessionCompleted || s.Status == SessionCancelled {
		return fmt.Errorf("cannot cancel a session that is %s", s.Status)
	}
	s.Status = SessionCancelled
	return nil
}

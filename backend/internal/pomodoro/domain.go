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

// NoiseConfig is a provision for future white noise integration.
// Currently unused — the fields are accepted at session creation but ignored.
type NoiseConfig struct {
	Enabled bool   `json:"enabled"`
	TrackID string `json:"track_id,omitempty"`
}

type PomodoroSession struct {
	ID            string
	Status        SessionStatus
	FocusDuration time.Duration // e.g. 25 minutes
	BreakDuration time.Duration // e.g. 5 minutes
	CurrentCycle  int           // which pomodoro cycle (1-based), for long break tracking
	StartedAt     time.Time     // shifted forward by pause duration on resume
	PausedAt      *time.Time    // set on pause, cleared on resume
	CompletedAt   *time.Time
	NoiseConfig   *NoiseConfig // provision for white noise
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
		ID:            id,
		Status:        SessionRunning,
		FocusDuration: time.Duration(focusMinutes) * time.Minute,
		BreakDuration: time.Duration(breakMinutes) * time.Minute,
		CurrentCycle:  1,
		StartedAt:     now,
		NoiseConfig:   noise,
	}, nil
}

func (s *PomodoroSession) Elapsed() time.Duration {
	if s.PausedAt != nil {
		return s.PausedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
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
	s.PausedAt = &now
	s.Status = SessionPaused
	return nil
}

func (s *PomodoroSession) Resume() error {
	if s.Status != SessionPaused {
		return fmt.Errorf("cannot resume a session that is %s", s.Status)
	}
	now := time.Now().UTC()
	// Shift StartedAt forward by the pause duration so Elapsed() is continuous.
	s.StartedAt = now.Add(-s.PausedAt.Sub(s.StartedAt))
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
		// Freeze elapsed at this moment by setting PausedAt.
		s.PausedAt = &now
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

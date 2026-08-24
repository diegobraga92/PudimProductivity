package pomodoro

import (
	"testing"
	"time"
)

func TestNewSessionDefaults(t *testing.T) {
	s, err := NewSession("s1", 0, 0, false, nil)
	if err != nil {
		t.Fatalf("NewSession returned error: %v", err)
	}
	if s.Phase != PhaseFocus {
		t.Errorf("expected phase focus, got %s", s.Phase)
	}
	if s.FocusDuration != 25*time.Minute {
		t.Errorf("expected focus duration 25m, got %s", s.FocusDuration)
	}
	if s.BreakDuration != 5*time.Minute {
		t.Errorf("expected break duration 5m, got %s", s.BreakDuration)
	}
	if s.CurrentCycle != 1 {
		t.Errorf("expected cycle 1, got %d", s.CurrentCycle)
	}
	if s.Status != SessionRunning {
		t.Errorf("expected running status, got %s", s.Status)
	}
	if s.Continuous {
		t.Error("expected continuous=false by default")
	}
}

func TestNewSessionRequiresID(t *testing.T) {
	if _, err := NewSession("", 25, 5, false, nil); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestSegmentDuration(t *testing.T) {
	s := &PomodoroSession{Phase: PhaseFocus, FocusDuration: 25 * time.Minute, BreakDuration: 5 * time.Minute}
	if got := s.SegmentDuration(); got != 25*time.Minute {
		t.Errorf("focus segment duration = %s, want 25m", got)
	}
	s.Phase = PhaseBreak
	if got := s.SegmentDuration(); got != 5*time.Minute {
		t.Errorf("break segment duration = %s, want 5m", got)
	}
}

func TestRemainingIsPhaseAware(t *testing.T) {
	now := time.Now().UTC()
	s := &PomodoroSession{
		Status:             SessionPaused,
		Phase:              PhaseBreak,
		FocusDuration:      25 * time.Minute,
		BreakDuration:      5 * time.Minute,
		AccumulatedElapsed: 2 * time.Minute,
		StartedAt:          now,
	}
	if got := s.Remaining(); got != 3*time.Minute {
		t.Errorf("remaining = %s, want 3m", got)
	}
}

func TestRemainingClampsAtZero(t *testing.T) {
	s := &PomodoroSession{
		Status:             SessionPaused,
		Phase:              PhaseFocus,
		FocusDuration:      time.Minute,
		AccumulatedElapsed: 2 * time.Minute,
	}
	if got := s.Remaining(); got != 0 {
		t.Errorf("remaining = %s, want 0", got)
	}
}

func TestNextSegmentFocusToBreak(t *testing.T) {
	started := time.Now().UTC()
	s := &PomodoroSession{
		ID:            "focus-1",
		UserID:        "u1",
		Status:        SessionRunning,
		Phase:         PhaseFocus,
		Continuous:    true,
		FocusDuration: 25 * time.Minute,
		BreakDuration: 5 * time.Minute,
		CurrentCycle:  1,
		StartedAt:     started,
	}

	next, err := s.NextSegment("break-1", started.Add(25*time.Minute))
	if err != nil {
		t.Fatalf("NextSegment returned error: %v", err)
	}
	if next.Phase != PhaseBreak {
		t.Errorf("expected break phase, got %s", next.Phase)
	}
	if next.CurrentCycle != 1 {
		t.Errorf("expected cycle to stay 1, got %d", next.CurrentCycle)
	}
	if next.ID != "break-1" {
		t.Errorf("expected new segment id, got %s", next.ID)
	}
	if next.UserID != "u1" {
		t.Errorf("expected user to carry over, got %s", next.UserID)
	}
	if !next.Continuous {
		t.Error("expected continuous flag to carry over")
	}
	if next.Status != SessionRunning {
		t.Errorf("expected running status, got %s", next.Status)
	}
}

func TestNextSegmentBreakToFocus(t *testing.T) {
	started := time.Now().UTC()
	s := &PomodoroSession{
		ID:            "break-1",
		UserID:        "u1",
		Status:        SessionRunning,
		Phase:         PhaseBreak,
		Continuous:    true,
		FocusDuration: 25 * time.Minute,
		BreakDuration: 5 * time.Minute,
		CurrentCycle:  1,
		StartedAt:     started,
	}

	next, err := s.NextSegment("focus-2", started.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("NextSegment returned error: %v", err)
	}
	if next.Phase != PhaseFocus {
		t.Errorf("expected focus phase, got %s", next.Phase)
	}
	if next.CurrentCycle != 2 {
		t.Errorf("expected cycle 2, got %d", next.CurrentCycle)
	}
}

func TestNextSegmentRequiresID(t *testing.T) {
	s := &PomodoroSession{Phase: PhaseFocus}
	if _, err := s.NextSegment("", time.Now().UTC()); err == nil {
		t.Fatal("expected error for empty id")
	}
}

package pomodoro

import (
	"testing"
	"time"
)

func TestToSessionResponseIncludesPhaseAndContinuous(t *testing.T) {
	now := time.Now().UTC()
	s := &PomodoroSession{
		ID:            "s1",
		Status:        SessionRunning,
		Phase:         PhaseBreak,
		Continuous:    true,
		FocusDuration: 25 * time.Minute,
		BreakDuration: 5 * time.Minute,
		CurrentCycle:  2,
		StartedAt:     now,
	}

	resp := toSessionResponse(s)
	if resp.Phase != "break" {
		t.Errorf("expected phase 'break', got %q", resp.Phase)
	}
	if !resp.Continuous {
		t.Error("expected continuous=true")
	}
	if resp.FocusDuration != 25 || resp.BreakDuration != 5 {
		t.Errorf("unexpected durations: focus=%d break=%d", resp.FocusDuration, resp.BreakDuration)
	}
	if resp.CurrentCycle != 2 {
		t.Errorf("expected cycle 2, got %d", resp.CurrentCycle)
	}
	if resp.Status != string(SessionRunning) {
		t.Errorf("expected status running, got %q", resp.Status)
	}
}

func TestToSessionResponseRemainingUsesSegmentDuration(t *testing.T) {
	now := time.Now().UTC()
	s := &PomodoroSession{
		ID:                 "s1",
		Status:             SessionPaused,
		Phase:              PhaseBreak,
		Continuous:         true,
		FocusDuration:      25 * time.Minute,
		BreakDuration:      5 * time.Minute,
		CurrentCycle:       1,
		AccumulatedElapsed: 2 * time.Minute,
		StartedAt:          now,
	}

	resp := toSessionResponse(s)
	if resp.RemainingSeconds != 180 {
		t.Errorf("expected remaining 180s (break segment), got %d", resp.RemainingSeconds)
	}
}

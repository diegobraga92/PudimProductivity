package insights

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

type stubRepo struct {
	report      *InsightReport
	sessions    []PomodoroSessionRecord
	completions []CompletionRecord
	recipes     int
}

func (r *stubRepo) SaveReport(_ context.Context, report *InsightReport) error {
	r.report = report
	return nil
}
func (r *stubRepo) GetReport(_ context.Context, _ string, _ time.Time) (*InsightReport, error) {
	return r.report, nil
}
func (r *stubRepo) UpsertSession(_ context.Context, session *PomodoroSessionRecord) error {
	r.sessions = append(r.sessions, *session)
	return nil
}
func (r *stubRepo) ListSessions(_ context.Context, _ string, _, _ time.Time) ([]PomodoroSessionRecord, error) {
	return r.sessions, nil
}
func (r *stubRepo) ListCompletions(_ context.Context, _, _ time.Time) ([]CompletionRecord, error) {
	return r.completions, nil
}
func (r *stubRepo) CountRecipes(_ context.Context, _, _ time.Time) (int, error) {
	return r.recipes, nil
}

func TestStartOfWeek_Monday(t *testing.T) {
	// 2026-08-10 is a Monday.
	monday := startOfWeek(time.Date(2026, 8, 10, 15, 30, 0, 0, time.UTC))
	if got, want := monday.Format("2006-01-02"), "2026-08-10"; got != want {
		t.Errorf("Monday of week: got %s, want %s", got, want)
	}
	// Sunday 2026-08-09 belongs to the previous week.
	sunday := startOfWeek(time.Date(2026, 8, 9, 23, 0, 0, 0, time.UTC))
	if got, want := sunday.Format("2006-01-02"), "2026-08-03"; got != want {
		t.Errorf("Monday of week for Sunday: got %s, want %s", got, want)
	}
}

func TestGenerateWeeklyReport_AggregatesAndRenders(t *testing.T) {
	repo := &stubRepo{
		completions: []CompletionRecord{
			{TaskID: "t1", Title: "Meditate", CompletedDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)},
			{TaskID: "t1", Title: "Meditate", CompletedDate: time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)},
			{TaskID: "t2", Title: "Read", CompletedDate: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)},
		},
		sessions: []PomodoroSessionRecord{
			{FocusMinutes: 25, ElapsedS: 1500, CompletedAt: time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)},
			{FocusMinutes: 25, ElapsedS: 1490, CompletedAt: time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)},
		},
		recipes: 2,
	}
	svc := NewInsightService(repo, nil, nil, nil)

	report, err := svc.GenerateWeeklyReport(context.Background(), "dev-user", time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GenerateWeeklyReport: %v", err)
	}

	if report.Stats.TotalCompletions != 3 {
		t.Errorf("TotalCompletions: got %d, want 3", report.Stats.TotalCompletions)
	}
	if report.Stats.FocusMinutes != 50 {
		t.Errorf("FocusMinutes: got %d, want 50", report.Stats.FocusMinutes)
	}
	if report.Stats.FocusSessions != 2 {
		t.Errorf("FocusSessions: got %d, want 2", report.Stats.FocusSessions)
	}
	if report.Stats.RecipesCreated != 2 {
		t.Errorf("RecipesCreated: got %d, want 2", report.Stats.RecipesCreated)
	}
	if len(report.Stats.TopHabits) != 2 || report.Stats.TopHabits[0].Title != "Meditate" || report.Stats.TopHabits[0].Count != 2 {
		t.Errorf("TopHabits: unexpected %+v", report.Stats.TopHabits)
	}
	if report.Stats.CompletionsPerDay != 3.0/7.0 {
		t.Errorf("CompletionsPerDay: got %v, want %v", report.Stats.CompletionsPerDay, 3.0/7.0)
	}
	if report.ReportText == "" {
		t.Error("ReportText: expected non-empty template output")
	}
	if report.Stats.WeekStart != "2026-08-10" {
		t.Errorf("WeekStart: got %s, want 2026-08-10", report.Stats.WeekStart)
	}
}

func TestGenerateWeeklyReport_UsesCache(t *testing.T) {
	repo := &stubRepo{}
	svc := NewInsightService(repo, nil, nil, nil)

	week := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	first, err := svc.GenerateWeeklyReport(context.Background(), "dev-user", week)
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}
	if repo.report == nil {
		t.Fatal("expected report to be saved")
	}

	// A second call must return the cached copy (no re-computation).
	second, err := svc.GenerateWeeklyReport(context.Background(), "dev-user", week)
	if err != nil {
		t.Fatalf("second generate: %v", err)
	}
	if second.ID != first.ID {
		t.Errorf("expected cached report (same ID), got new %s vs %s", second.ID, first.ID)
	}
}

func TestSessionRecordFromEvent_MalformedReturnsNil(t *testing.T) {
	if got := sessionRecordFromEvent(eventbus.Event{Payload: "not-a-map"}); got != nil {
		t.Error("non-map payload should return nil")
	}
	if got := sessionRecordFromEvent(eventbus.Event{Payload: map[string]any{"session_id": ""}}); got != nil {
		t.Error("missing session_id should return nil")
	}
}

func TestSessionRecordFromEvent_CoercesInts(t *testing.T) {
	// The pomodoro module publishes Go ints; JSON-decoded maps produce
	// float64/json.Number. Both must coerce to the record's int fields.
	ev := eventbus.Event{
		Payload: map[string]any{
			"session_id":    "s1",
			"user_id":       "dev-user",
			"focus_minutes": 25, // int
			"elapsed_s":     json.Number("1499"),
			"started_at":    "2026-08-10T10:00:00Z",
			"completed_at":  "2026-08-10T10:25:00Z",
		},
	}
	rec := sessionRecordFromEvent(ev)
	if rec == nil {
		t.Fatal("expected a record")
	}
	if rec.FocusMinutes != 25 {
		t.Errorf("FocusMinutes: got %d, want 25", rec.FocusMinutes)
	}
	if rec.ElapsedS != 1499 {
		t.Errorf("ElapsedS: got %d, want 1499", rec.ElapsedS)
	}
	if rec.UserID != "dev-user" {
		t.Errorf("UserID: got %q", rec.UserID)
	}
}

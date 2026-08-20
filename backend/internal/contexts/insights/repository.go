package insights

import (
	"context"
	"time"
)

// InsightRepository persists reports + pomodoro sessions and reads the raw
// data the report aggregates.
type InsightRepository interface {
	// SaveReport upserts a weekly report (unique by user_id + week_start).
	SaveReport(ctx context.Context, report *InsightReport) error

	// GetReport returns the cached report for a week, or nil when absent.
	GetReport(ctx context.Context, userID string, weekStart time.Time) (*InsightReport, error)

	// UpsertSession persists a completed pomodoro session (idempotent by id).
	UpsertSession(ctx context.Context, session *PomodoroSessionRecord) error

	// ListSessions returns completed focus sessions in [from, to).
	ListSessions(ctx context.Context, userID string, from, to time.Time) ([]PomodoroSessionRecord, error)

	// ListCompletions returns task completions in [from, to) with task titles.
	// The dev data model is effectively single-user; all completions are
	// returned (see ADR 011).
	ListCompletions(ctx context.Context, from, to time.Time) ([]CompletionRecord, error)

	// CountRecipes returns recipes created in [from, to).
	CountRecipes(ctx context.Context, from, to time.Time) (int, error)
}

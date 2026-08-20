package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/insights"
)

type InsightsRepository struct {
	pool *pgxpool.Pool
}

func NewInsightsRepository(pool *pgxpool.Pool) *InsightsRepository {
	return &InsightsRepository{pool: pool}
}

func (r *InsightsRepository) SaveReport(ctx context.Context, report *insights.InsightReport) error {
	statsJSON, err := json.Marshal(report.Stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO insight_reports (id, user_id, week_start, report_json, report_text, llm_summary, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, week_start) DO UPDATE
		SET report_json = EXCLUDED.report_json,
		    report_text = EXCLUDED.report_text,
		    llm_summary = EXCLUDED.llm_summary,
		    created_at  = EXCLUDED.created_at
	`,
		report.ID, report.UserID, report.WeekStart, statsJSON, report.ReportText, report.LLMSummary, report.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save insight report: %w", err)
	}
	return nil
}

func (r *InsightsRepository) GetReport(ctx context.Context, userID string, weekStart time.Time) (*insights.InsightReport, error) {
	var report insights.InsightReport
	var statsJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, week_start, report_json, report_text, llm_summary, created_at
		FROM insight_reports
		WHERE user_id = $1 AND week_start = $2
	`, userID, weekStart).Scan(
		&report.ID, &report.UserID, &report.WeekStart, &statsJSON, &report.ReportText, &report.LLMSummary, &report.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get insight report: %w", err)
	}

	if err := json.Unmarshal(statsJSON, &report.Stats); err != nil {
		return nil, fmt.Errorf("unmarshal report stats: %w", err)
	}
	return &report, nil
}

func (r *InsightsRepository) UpsertSession(ctx context.Context, session *insights.PomodoroSessionRecord) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pomodoro_sessions (id, user_id, focus_minutes, elapsed_s, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`,
		session.ID, session.UserID, session.FocusMinutes, session.ElapsedS, session.StartedAt, session.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert pomodoro session: %w", err)
	}
	return nil
}

func (r *InsightsRepository) ListSessions(ctx context.Context, userID string, from, to time.Time) ([]insights.PomodoroSessionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, focus_minutes, elapsed_s, started_at, completed_at
		FROM pomodoro_sessions
		WHERE user_id = $1 AND completed_at >= $2 AND completed_at < $3
		ORDER BY completed_at ASC
	`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list pomodoro sessions: %w", err)
	}
	defer rows.Close()

	var sessions []insights.PomodoroSessionRecord
	for rows.Next() {
		var s insights.PomodoroSessionRecord
		if err := rows.Scan(&s.ID, &s.UserID, &s.FocusMinutes, &s.ElapsedS, &s.StartedAt, &s.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan pomodoro session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pomodoro sessions: %w", err)
	}
	if sessions == nil {
		sessions = make([]insights.PomodoroSessionRecord, 0)
	}
	return sessions, nil
}

func (r *InsightsRepository) ListCompletions(ctx context.Context, from, to time.Time) ([]insights.CompletionRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.task_id, COALESCE(t.title, 'Unknown'), c.completed_date
		FROM task_completions c
		LEFT JOIN tasks t ON t.id = c.task_id
		WHERE c.completed_date >= $1 AND c.completed_date < $2 AND c.deleted_at IS NULL
		ORDER BY c.completed_date ASC
	`, from, to)
	if err != nil {
		return nil, fmt.Errorf("list completions: %w", err)
	}
	defer rows.Close()

	var completions []insights.CompletionRecord
	for rows.Next() {
		var c insights.CompletionRecord
		if err := rows.Scan(&c.TaskID, &c.Title, &c.CompletedDate); err != nil {
			return nil, fmt.Errorf("scan completion: %w", err)
		}
		completions = append(completions, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate completions: %w", err)
	}
	if completions == nil {
		completions = make([]insights.CompletionRecord, 0)
	}
	return completions, nil
}

func (r *InsightsRepository) CountRecipes(ctx context.Context, from, to time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM recipes
		WHERE created_at >= $1 AND created_at < $2
	`, from, to).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count recipes: %w", err)
	}
	return count, nil
}

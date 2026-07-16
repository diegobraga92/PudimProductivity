package planner

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPlannerRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPlannerRepository(pool *pgxpool.Pool) *PostgresPlannerRepository {
	return &PostgresPlannerRepository{pool: pool}
}

func (r *PostgresPlannerRepository) Create(ctx context.Context, entry *PlannerEntry) error {
	query := `
		INSERT INTO planner_entries (id, title, days, start_time, end_time, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		entry.ID,
		entry.Title,
		entry.Days,
		entry.StartTime,
		entry.EndTime,
		entry.Color,
		entry.CreatedAt,
		entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert planner entry: %w", err)
	}

	return nil
}

func (r *PostgresPlannerRepository) GetByID(ctx context.Context, id string) (*PlannerEntry, error) {
	query := `
		SELECT id, title, days, start_time, end_time, color, created_at, updated_at
		FROM planner_entries
		WHERE id = $1
	`

	entry := &PlannerEntry{}

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&entry.ID,
		&entry.Title,
		&entry.Days,
		&entry.StartTime,
		&entry.EndTime,
		&entry.Color,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPlannerEntryNotFound
		}
		return nil, fmt.Errorf("get planner entry by id: %w", err)
	}

	return entry, nil
}

func (r *PostgresPlannerRepository) List(ctx context.Context) ([]*PlannerEntry, error) {
	query := `
		SELECT id, title, days, start_time, end_time, color, created_at, updated_at
		FROM planner_entries
		ORDER BY start_time ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list planner entries: %w", err)
	}
	defer rows.Close()

	var entries []*PlannerEntry
	for rows.Next() {
		entry := &PlannerEntry{}

		err := rows.Scan(
			&entry.ID,
			&entry.Title,
			&entry.Days,
			&entry.StartTime,
			&entry.EndTime,
			&entry.Color,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan planner entry row: %w", err)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate planner entry rows: %w", err)
	}

	if entries == nil {
		entries = make([]*PlannerEntry, 0)
	}

	return entries, nil
}

func (r *PostgresPlannerRepository) Update(ctx context.Context, entry *PlannerEntry) error {
	query := `
		UPDATE planner_entries
		SET title = $1, days = $2, start_time = $3, end_time = $4, color = $5, updated_at = $6
		WHERE id = $7
	`

	result, err := r.pool.Exec(ctx, query,
		entry.Title,
		entry.Days,
		entry.StartTime,
		entry.EndTime,
		entry.Color,
		entry.UpdatedAt,
		entry.ID,
	)
	if err != nil {
		return fmt.Errorf("update planner entry: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPlannerEntryNotFound
	}

	return nil
}

func (r *PostgresPlannerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM planner_entries WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete planner entry: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPlannerEntryNotFound
	}

	return nil
}

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// returns the full set of columns used in SELECT, INSERT, and UPDATE queries.
const taskColumns = `id, title, status, recurrence_days, list_id, start_time, end_time, color, scheduled_date, alarm_minutes, updated_by, created_at, updated_at, position`

func (r *TaskRepository) scanTask(scanner interface {
	Scan(dest ...any) error
}) (*taskdomain.Task, error) {
	task := &taskdomain.Task{}

	// pgx cannot scan DATE/TIME into *string directly in binary mode.
	// Use intermediate variables for time/date columns.
	var startTime, endTime *string
	var color *string
	var scheduledDate *time.Time
	var alarmMinutes *int

	err := scanner.Scan(
		&task.ID,
		&task.Title,
		(*string)(&task.Status),
		&task.RecurrenceDays,
		&task.ListID,
		&startTime,
		&endTime,
		&color,
		&scheduledDate,
		&alarmMinutes,
		&task.UpdatedBy,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.Position,
	)
	if err != nil {
		return nil, err
	}

	// Copy scalar values back
	task.StartTime = startTime
	task.EndTime = endTime
	task.Color = color
	if scheduledDate != nil {
		s := scheduledDate.Format("2006-01-02")
		task.ScheduledDate = &s
	}
	task.AlarmMinutes = alarmMinutes

	return task, nil
}

func (r *TaskRepository) Create(ctx context.Context, task *taskdomain.Task) error {
	// New tasks take the first rank of their bucket and the rest shifts down.
	// One statement keeps the shift and the insert atomic for concurrent creates.
	query := `
		WITH shifted AS (
			UPDATE tasks
			SET position = position + 1
			WHERE deleted_at IS NULL
			  AND list_id IS NOT DISTINCT FROM $5
			  AND (recurrence_days IS NOT NULL) = ($4::text[] IS NOT NULL)
			  AND status = $3
		)
		INSERT INTO tasks (` + taskColumns + `)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 0)
		RETURNING position
	`

	err := r.pool.QueryRow(ctx, query,
		task.ID,
		task.Title,
		string(task.Status),
		task.RecurrenceDays,
		task.ListID,
		task.StartTime,
		task.EndTime,
		task.Color,
		task.ScheduledDate,
		task.AlarmMinutes,
		task.UpdatedBy,
		task.CreatedAt,
		task.UpdatedAt,
	).Scan(&task.Position)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (*taskdomain.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1 AND deleted_at IS NULL`

	task, err := r.scanTask(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, taskdomain.ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) List(ctx context.Context, statusFilter, typeFilter string) ([]*taskdomain.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE list_id IS NULL AND deleted_at IS NULL`
	args := make([]any, 0)

	if statusFilter != "" {
		args = append(args, statusFilter)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}

	if typeFilter != "" {
		switch typeFilter {
		case "one-off":
			query += " AND recurrence_days IS NULL"
		case "habit":
			query += " AND recurrence_days IS NOT NULL"
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*taskdomain.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}

	if tasks == nil {
		tasks = make([]*taskdomain.Task, 0)
	}

	return tasks, nil
}

func (r *TaskRepository) ListScheduled(ctx context.Context) ([]*taskdomain.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE start_time IS NOT NULL AND deleted_at IS NULL AND list_id IS NULL ORDER BY start_time ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*taskdomain.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan scheduled task row: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scheduled task rows: %w", err)
	}

	if tasks == nil {
		tasks = make([]*taskdomain.Task, 0)
	}

	return tasks, nil
}

func (r *TaskRepository) ListByListID(ctx context.Context, listID, typeFilter string) ([]*taskdomain.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE list_id = $1 AND deleted_at IS NULL`
	args := []any{listID}

	if typeFilter != "" {
		switch typeFilter {
		case "one-off":
			query += " AND recurrence_days IS NULL"
		case "habit":
			query += " AND recurrence_days IS NOT NULL"
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks by list id: %w", err)
	}
	defer rows.Close()

	var tasks []*taskdomain.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task rows: %w", err)
	}

	if tasks == nil {
		tasks = make([]*taskdomain.Task, 0)
	}

	return tasks, nil
}

func (r *TaskRepository) NextPosition(ctx context.Context, scope taskdomain.TaskScope) (int, error) {
	query := `
		SELECT COALESCE(MAX(position) + 1, 0)
		FROM tasks
		WHERE deleted_at IS NULL
		  AND list_id IS NOT DISTINCT FROM $1
		  AND (recurrence_days IS NOT NULL) = $2
		  AND status = $3
	`

	var position int
	if err := r.pool.QueryRow(ctx, query, scope.ListID, scope.IsHabit, string(scope.Status)).Scan(&position); err != nil {
		return 0, fmt.Errorf("next task position: %w", err)
	}

	return position, nil
}

func (r *TaskRepository) Reorder(ctx context.Context, scope taskdomain.TaskScope, orderedIDs []string) ([]*taskdomain.Task, error) {
	positions := make([]int32, len(orderedIDs))
	for i := range positions {
		positions[i] = int32(i)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin reorder tasks: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// One statement, so the ranks in a bucket never end up half applied. The
	// scope predicates are repeated here on purpose: they are what marks an id
	// as out of scope, including ids deleted since the client loaded the list.
	result, err := tx.Exec(ctx, `
		WITH ordered AS (
			SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS position
		)
		UPDATE tasks
		SET position = ordered.position, updated_at = NOW()
		FROM ordered
		WHERE tasks.id = ordered.id
		  AND tasks.deleted_at IS NULL
		  AND tasks.list_id IS NOT DISTINCT FROM $3
		  AND (tasks.recurrence_days IS NOT NULL) = $4
		  AND tasks.status = $5
	`, orderedIDs, positions, scope.ListID, scope.IsHabit, string(scope.Status))
	if err != nil {
		return nil, fmt.Errorf("update task positions: %w", err)
	}

	if int(result.RowsAffected()) != len(orderedIDs) {
		return nil, taskdomain.ErrTaskScopeMismatch
	}

	rows, err := tx.Query(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL`, orderedIDs)
	if err != nil {
		return nil, fmt.Errorf("load reordered tasks: %w", err)
	}

	tasks := make([]*taskdomain.Task, 0, len(orderedIDs))
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan reordered task row: %w", err)
		}
		tasks = append(tasks, task)
	}
	rows.Close()

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reordered task rows: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit reorder tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *taskdomain.Task) error {
	query := `
		UPDATE tasks
		SET title = $1, status = $2, recurrence_days = $3, list_id = $4,
		    start_time = $5, end_time = $6, color = $7, scheduled_date = $8,
		    alarm_minutes = $9, updated_by = $10, updated_at = $11
		WHERE id = $12
	`

	result, err := r.pool.Exec(ctx, query,
		task.Title,
		string(task.Status),
		task.RecurrenceDays,
		task.ListID,
		task.StartTime,
		task.EndTime,
		task.Color,
		task.ScheduledDate,
		task.AlarmMinutes,
		task.UpdatedBy,
		task.UpdatedAt,
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrTaskNotFound
	}

	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	// Soft delete so offline clients can learn about the deletion
	// through the incremental sync endpoint.
	query := `UPDATE tasks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrTaskNotFound
	}

	return nil
}

func (r *TaskRepository) CreateCompletion(ctx context.Context, completion *taskdomain.TaskCompletion) error {
	// A soft-deleted tombstone no longer blocks re-completing the same date after an uncomplete,
	// while a genuinely active completion still yields taskdomain.ErrCompletionAlreadyExists.
	query := `
		INSERT INTO task_completions (id, task_id, completed_date, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, completed_date) WHERE deleted_at IS NULL DO NOTHING
	`

	result, err := r.pool.Exec(ctx, query,
		completion.ID,
		completion.TaskID,
		completion.CompletedDate,
		completion.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task completion: %w", err)
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrCompletionAlreadyExists
	}

	return nil
}

func (r *TaskRepository) DeleteCompletion(ctx context.Context, taskID string, date time.Time) error {
	query := `UPDATE task_completions SET deleted_at = NOW() WHERE task_id = $1 AND completed_date = $2 AND deleted_at IS NULL`

	result, err := r.pool.Exec(ctx, query, taskID, date)
	if err != nil {
		return fmt.Errorf("delete task completion: %w", err)
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrCompletionNotFound
	}

	return nil
}

func (r *TaskRepository) GetCompletion(ctx context.Context, taskID string, date time.Time) (*taskdomain.TaskCompletion, error) {
	query := `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE task_id = $1 AND completed_date = $2 AND deleted_at IS NULL
	`

	completion := &taskdomain.TaskCompletion{}

	err := r.pool.QueryRow(ctx, query, taskID, date).Scan(
		&completion.ID,
		&completion.TaskID,
		&completion.CompletedDate,
		&completion.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get task completion: %w", err)
	}

	return completion, nil
}

func (r *TaskRepository) ListAllCompletions(ctx context.Context, from, to time.Time) ([]*taskdomain.TaskCompletion, error) {
	query := `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE completed_date >= $1 AND completed_date <= $2 AND deleted_at IS NULL
		ORDER BY task_id, completed_date ASC
	`

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("list all task completions: %w", err)
	}
	defer rows.Close()

	var completions []*taskdomain.TaskCompletion
	for rows.Next() {
		c := &taskdomain.TaskCompletion{}

		err := rows.Scan(
			&c.ID,
			&c.TaskID,
			&c.CompletedDate,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task completion row: %w", err)
		}

		completions = append(completions, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task completion rows: %w", err)
	}

	if completions == nil {
		completions = make([]*taskdomain.TaskCompletion, 0)
	}

	return completions, nil
}

func (r *TaskRepository) ListCompletions(ctx context.Context, taskID string, from, to time.Time) ([]*taskdomain.TaskCompletion, error) {
	query := `
		SELECT id, task_id, completed_date, created_at
		FROM task_completions
		WHERE task_id = $1 AND completed_date >= $2 AND completed_date <= $3 AND deleted_at IS NULL
		ORDER BY completed_date ASC
	`

	rows, err := r.pool.Query(ctx, query, taskID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list task completions: %w", err)
	}
	defer rows.Close()

	var completions []*taskdomain.TaskCompletion
	for rows.Next() {
		c := &taskdomain.TaskCompletion{}

		err := rows.Scan(
			&c.ID,
			&c.TaskID,
			&c.CompletedDate,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task completion row: %w", err)
		}

		completions = append(completions, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task completion rows: %w", err)
	}

	if completions == nil {
		completions = make([]*taskdomain.TaskCompletion, 0)
	}

	return completions, nil
}

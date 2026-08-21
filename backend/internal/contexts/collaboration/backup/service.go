package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Sentinel errors returned by Service.Import.
var (
	ErrInvalidBackup      = errors.New("invalid backup document")
	ErrUnsupportedVersion = errors.New("unsupported backup version")
)

type backupTable struct {
	name     string // table name
	idColumn string // column used to order rows for deterministic exports
	columns  string // e.g. "id uuid, title text, status text"
}

func (t backupTable) columnNames() string {
	parts := strings.Split(t.columns, ",")
	names := make([]string, len(parts))
	for i, p := range parts {
		fields := strings.Fields(strings.TrimSpace(p))
		names[i] = fields[0]
	}
	return strings.Join(names, ", ")
}

// TODO: Check tables after migrations are finalized

// backupTables lists every table included in a backup, in dependency order,
// parents before children, so foreign keys hold during import.
var backupTables = []backupTable{
	{
		name:     "feature_flags",
		idColumn: "id",
		columns:  "id uuid, name text, description text, enabled boolean, created_at timestamptz, updated_at timestamptz",
	},
	{
		name:     "task_lists",
		idColumn: "id",
		columns:  "id uuid, name text, description text, created_at timestamptz, updated_at timestamptz, owner_id text, deleted_at timestamptz",
	},
	{
		name:     "tasks",
		idColumn: "id",
		columns:  "id uuid, title text, status text, created_at timestamptz, updated_at timestamptz, recurrence_days text[], list_id uuid, start_time time, end_time time, color text, scheduled_date date, alarm_minutes integer, updated_by text, deleted_at timestamptz",
	},
	{
		name:     "task_completions",
		idColumn: "id",
		columns:  "id uuid, task_id uuid, completed_date date, created_at timestamptz, deleted_at timestamptz",
	},
	{
		name:     "task_list_shares",
		idColumn: "list_id",
		columns:  "list_id uuid, shared_with text, role text, created_at timestamptz, deleted_at timestamptz",
	},
	{
		name:     "planner_entries",
		idColumn: "id",
		columns:  "id uuid, title text, days text[], start_time time, end_time time, color text, created_at timestamptz, updated_at timestamptz",
	},
	{
		name:     "library_items",
		idColumn: "id",
		columns:  "id uuid, name text, media_type text, release_year integer, done boolean, notes text, created_at timestamptz, updated_at timestamptz",
	},
	{
		name:     "recipes",
		idColumn: "id",
		columns:  "id uuid, title text, description text, difficulty text, prep_time_minutes integer, cook_time_minutes integer, servings integer, image_url text, source_url text, created_at timestamptz, updated_at timestamptz",
	},
	{
		name:     "recipe_tags",
		idColumn: "recipe_id",
		columns:  "recipe_id uuid, tag text",
	},
	{
		name:     "recipe_ingredients",
		idColumn: "id",
		columns:  "id uuid, recipe_id uuid, name text, quantity text, unit text, sort_order integer",
	},
	{
		name:     "recipe_steps",
		idColumn: "id",
		columns:  "id uuid, recipe_id uuid, step_number integer, instruction text",
	},
	{
		name:     "pomodoro_sessions",
		idColumn: "id",
		columns:  "id uuid, user_id text, focus_minutes integer, elapsed_s integer, started_at timestamptz, completed_at timestamptz",
	},
	{
		name:     "insight_reports",
		idColumn: "id",
		columns:  "id uuid, user_id text, week_start date, report_json jsonb, report_text text, llm_summary text, created_at timestamptz",
	},
}

// Service exports and restores the non-sensitive application data.
type Service struct {
	pool *pgxpool.Pool
}

// NewService constructs a Service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Export snapshots every backup table into a single BackupFile and returns it
// serialized as indented JSON.
func (s *Service) Export(ctx context.Context, appVersion string) ([]byte, error) {
	backup := BackupFile{
		Version:    BackupFormatVersion,
		ExportedAt: time.Now().UTC(),
		AppVersion: appVersion,
		RowCounts:  make(map[string]int, len(backupTables)),
		Tables:     make(map[string]json.RawMessage, len(backupTables)),
	}

	for _, table := range backupTables {
		rows, err := s.readTable(ctx, table)
		if err != nil {
			return nil, fmt.Errorf("export table %s: %w", table.name, err)
		}
		backup.Tables[table.name] = rows
		backup.RowCounts[table.name] = jsonArrayLen(rows)
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal backup: %w", err)
	}
	log.Info().Int64("bytes", int64(len(data))).Msg("backup exported")
	return data, nil
}

// TODO: Check about using json_array_length to count lines, instead of doing unmarshall in export and import

// readTable returns all rows of a table as a JSON array.
func (s *Service) readTable(ctx context.Context, table backupTable) (json.RawMessage, error) {
	q := fmt.Sprintf(
		`SELECT COALESCE(json_agg(row_to_json(t) ORDER BY t.%s), '[]'::json) FROM %s t`,
		table.idColumn, table.name,
	)
	var raw json.RawMessage
	if err := s.pool.QueryRow(ctx, q).Scan(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// Import validates a backup and restores it into the database, replacing the current contents.
// On any error every change is rolled back and the pre-restore data is left untouched.
func (s *Service) Import(ctx context.Context, data []byte) (ImportResult, error) {
	var backup BackupFile
	if err := json.Unmarshal(data, &backup); err != nil {
		return ImportResult{}, fmt.Errorf("%w: %v", ErrInvalidBackup, err)
	}
	if backup.Version != BackupFormatVersion {
		return ImportResult{}, fmt.Errorf("%w %q (expected %q)", ErrUnsupportedVersion, backup.Version, BackupFormatVersion)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.truncateAll(ctx, tx); err != nil {
		return ImportResult{}, err
	}

	result := ImportResult{
		RestoredAt: time.Now().UTC(),
		RowCounts:  make(map[string]int, len(backupTables)),
	}

	for _, table := range backupTables {
		raw, ok := backup.Tables[table.name]
		if !ok || len(raw) == 0 {
			result.RowCounts[table.name] = 0
			continue
		}
		n, err := s.insertTable(ctx, tx, table, raw)
		if err != nil {
			return ImportResult{}, fmt.Errorf("restore table %s: %w", table.name, err)
		}
		result.RowCounts[table.name] = n
	}

	if err := tx.Commit(ctx); err != nil {
		return ImportResult{}, fmt.Errorf("commit restore: %w", err)
	}

	log.Info().Interface("row_counts", result.RowCounts).Msg("backup restore completed")
	return result, nil
}

// TODO: Check about adding timeout for contexts, this and any others

func (s *Service) truncateAll(ctx context.Context, tx pgx.Tx) error {
	names := make([]string, len(backupTables))
	for i, t := range backupTables {
		names[i] = t.name
	}
	q := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(names, ", "))
	if _, err := tx.Exec(ctx, q); err != nil {
		return fmt.Errorf("truncate tables: %w", err)
	}
	return nil
}

// TODO: Add log or similar in case jsonb_to_recordset discards additional columns, or nulls missing columns

func (s *Service) insertTable(ctx context.Context, tx pgx.Tx, table backupTable, raw json.RawMessage) (int, error) {
	names := table.columnNames()
	q := fmt.Sprintf(
		"INSERT INTO %s (%s) SELECT %s FROM jsonb_to_recordset($1::jsonb) AS t(%s)",
		table.name, names, names, table.columns,
	)
	if _, err := tx.Exec(ctx, q, raw); err != nil {
		return 0, err
	}
	return jsonArrayLen(raw), nil
}

func jsonArrayLen(raw json.RawMessage) int {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return 0
	}
	return len(arr)
}

package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/collaboration/backup"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres"
	"github.com/diegobraga92/pudimproductivity/backend/internal/infrastructure/postgres/postgrestest"
)

// seedBackupTestData inserts one row into every backed-up table so the test
// exercises the full round-trip (including arrays, dates and FKs).
func seedBackupTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	seed := `
		INSERT INTO task_lists (id, name, description) VALUES
			('11111111-1111-1111-1111-111111111111', 'Work', 'Ship things');
		INSERT INTO tasks (id, title, status, list_id, recurrence_days, start_time, end_time, scheduled_date, alarm_minutes, updated_by)
			VALUES ('22222222-2222-2222-2222-222222222222', 'Ship backup', 'done',
			        '11111111-1111-1111-1111-111111111111', ARRAY['mon','wed'], '09:00', '10:00', '2026-01-15', 30, 'dev-user');
		INSERT INTO task_completions (id, task_id, completed_date) VALUES
			('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', '2026-01-15');
		INSERT INTO planner_entries (id, title, days, start_time, end_time, color) VALUES
			('44444444-4444-4444-4444-444444444444', 'Deep work', ARRAY['mon'], '09:00', '10:00', '#3B82F6');
		INSERT INTO library_items (id, name, media_type, release_year, done, subtype, score, score_source) VALUES
			('55555555-5555-5555-5555-555555555555', 'Permanent Record', 'book', 2019, true, 'memoir', 8.5, 'imdb');
		INSERT INTO recipes (id, title, difficulty, prep_time_minutes, cook_time_minutes, servings) VALUES
			('66666666-6666-6666-6666-666666666666', 'Pancakes', 'easy', 5, 10, 2);
		INSERT INTO recipe_tags (recipe_id, tag) VALUES ('66666666-6666-6666-6666-666666666666', 'breakfast');
		INSERT INTO recipe_ingredients (id, recipe_id, name, quantity, unit, sort_order) VALUES
			('77777777-7777-7777-7777-777777777777', '66666666-6666-6666-6666-666666666666', 'Flour', '200', 'g', 1);
		INSERT INTO recipe_steps (id, recipe_id, step_number, instruction) VALUES
			('88888888-8888-8888-8888-888888888888', '66666666-6666-6666-6666-666666666666', 1, 'Mix and cook');
		INSERT INTO pomodoro_sessions (id, user_id, focus_minutes, elapsed_s, started_at, completed_at) VALUES
			('cccccccc-cccc-cccc-cccc-cccccccccccc', 'dev-user', 25, 1500, NOW(), NOW());
	`
	if _, err := pool.Exec(ctx, seed); err != nil {
		t.Fatalf("seed data: %v", err)
	}
}

func TestBackupRepository_ExportImportRoundTrip(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	seedBackupTestData(t, ctx, pool)
	store := postgres.NewBackupRepository(pool)

	data, err := store.Export(ctx, "test-version")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	var bf backup.BackupFile
	if err := json.Unmarshal(data, &bf); err != nil {
		t.Fatalf("unmarshal exported backup: %v", err)
	}
	if bf.Version != backup.BackupFormatVersion {
		t.Fatalf("version = %q, want %q", bf.Version, backup.BackupFormatVersion)
	}
	if bf.AppVersion != "test-version" {
		t.Fatalf("app_version = %q, want test-version", bf.AppVersion)
	}
	expectOne(t, bf.RowCounts, "tasks")
	expectOne(t, bf.RowCounts, "task_lists")
	expectOne(t, bf.RowCounts, "task_completions")
	expectOne(t, bf.RowCounts, "planner_entries")
	expectOne(t, bf.RowCounts, "library_items")
	expectOne(t, bf.RowCounts, "recipes")
	expectOne(t, bf.RowCounts, "recipe_tags")
	expectOne(t, bf.RowCounts, "recipe_ingredients")
	expectOne(t, bf.RowCounts, "recipe_steps")
	expectOne(t, bf.RowCounts, "pomodoro_sessions")
	if bf.RowCounts["feature_flags"] < 1 {
		t.Fatalf("feature_flags row count = %d, want >= 1 (seeded by migration)", bf.RowCounts["feature_flags"])
	}

	// 2. Simulate a lost database: wipe every table the backup contains.
	names := make([]string, 0, len(bf.Tables))
	for name := range bf.Tables {
		names = append(names, name)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+strings.Join(names, ", ")+" CASCADE"); err != nil {
		t.Fatalf("wipe tables: %v", err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM tasks").Scan(&remaining); err != nil {
		t.Fatalf("count after wipe: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("tasks not wiped: %d rows remain", remaining)
	}

	// 3. Restore from the backup.
	result, err := store.Import(ctx, data)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if result.RowCounts["tasks"] != 1 || result.RowCounts["recipes"] != 1 {
		t.Fatalf("restored row counts = %v", result.RowCounts)
	}

	// 4. Verify the restored rows round-tripped their values and types.
	var title string
	var listID string
	var recDays []string
	var schedDate time.Time
	var alarm *int
	err = pool.QueryRow(ctx,
		`SELECT title, list_id, recurrence_days, scheduled_date, alarm_minutes FROM tasks WHERE id = '22222222-2222-2222-2222-222222222222'`,
	).Scan(&title, &listID, &recDays, &schedDate, &alarm)
	if err != nil {
		t.Fatalf("query restored task: %v", err)
	}
	if title != "Ship backup" || listID != "11111111-1111-1111-1111-111111111111" ||
		len(recDays) != 2 || recDays[0] != "mon" || schedDate.Format("2006-01-02") != "2026-01-15" || alarm == nil || *alarm != 30 {
		t.Fatalf("restored task mismatch: title=%q list_id=%q days=%v date=%q alarm=%v", title, listID, recDays, schedDate.Format("2006-01-02"), alarm)
	}

	var ingName string
	var ingOrder int
	if err := pool.QueryRow(ctx,
		`SELECT name, sort_order FROM recipe_ingredients WHERE id = '77777777-7777-7777-7777-777777777777'`,
	).Scan(&ingName, &ingOrder); err != nil {
		t.Fatalf("query restored ingredient: %v", err)
	}
	if ingName != "Flour" || ingOrder != 1 {
		t.Fatalf("restored ingredient mismatch: name=%q sort_order=%d", ingName, ingOrder)
	}

	// Library items must round-trip their score columns (regression: they were
	// once missing from the backup column list and silently dropped on restore).
	var sub string
	var score *float64
	var scoreSource string
	if err := pool.QueryRow(ctx,
		`SELECT subtype, score, score_source FROM library_items WHERE id = '55555555-5555-5555-5555-555555555555'`,
	).Scan(&sub, &score, &scoreSource); err != nil {
		t.Fatalf("query restored library item: %v", err)
	}
	if sub != "memoir" || score == nil || *score != 8.5 || scoreSource != "imdb" {
		t.Fatalf("restored library item mismatch: subtype=%q score=%v score_source=%q", sub, score, scoreSource)
	}

	// 5. Excluded tables are untouched by a restore.
	var usersCount int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&usersCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if usersCount == 0 {
		t.Fatal("users table was wiped by restore — it must be excluded from TRUNCATE")
	}
}

func TestBackupRepository_ImportRejectsBadInput(t *testing.T) {
	postgrestest.SkipIfShort(t)
	ctx, pool := postgrestest.SetupPool(t)
	store := postgres.NewBackupRepository(pool)

	// Malformed JSON → ErrInvalidBackup, nothing touched.
	if _, err := store.Import(ctx, []byte(`{"tables": [`)); !errors.Is(err, backup.ErrInvalidBackup) {
		t.Fatalf("malformed JSON: want ErrInvalidBackup, got %v", err)
	}

	// Unknown version → ErrUnsupportedVersion, nothing touched.
	bad := []byte(`{"version":"99","tables":{}}`)
	if _, err := store.Import(ctx, bad); !errors.Is(err, backup.ErrUnsupportedVersion) {
		t.Fatalf("bad version: want ErrUnsupportedVersion, got %v", err)
	}
}

func expectOne(t *testing.T, counts map[string]int, table string) {
	t.Helper()
	if got := counts[table]; got != 1 {
		t.Fatalf("%s row count = %d, want 1", table, got)
	}
}

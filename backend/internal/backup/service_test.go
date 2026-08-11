package backup

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/diegobraga92/pudimproductivity/backend/internal/db"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

func setupBackupTestPostgres(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("pudimproductivity"),
		postgres.WithUsername("pudim"),
		postgres.WithPassword("pudim_dev"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := db.ConnectPool(ctx, shared.DatabaseConfig{
		URL:             connStr,
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("ConnectPool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.RunMigrations(ctx, pool); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	return ctx, pool
}

// seedBackupTestData inserts one row into every backed-up table so the test
// exercises the full round-trip (including arrays, jsonb, dates and FKs).
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
		INSERT INTO books (id, isbn, title, authors, status) VALUES
			('55555555-5555-5555-5555-555555555555', '9781250237231', 'Permanent Record', ARRAY['Edward Snowden'], 'reading');
		INSERT INTO recipes (id, title, difficulty, prep_time_minutes, cook_time_minutes, servings) VALUES
			('66666666-6666-6666-6666-666666666666', 'Pancakes', 'easy', 5, 10, 2);
		INSERT INTO recipe_tags (recipe_id, tag) VALUES ('66666666-6666-6666-6666-666666666666', 'breakfast');
		INSERT INTO recipe_ingredients (id, recipe_id, name, quantity, unit, sort_order) VALUES
			('77777777-7777-7777-7777-777777777777', '66666666-6666-6666-6666-666666666666', 'Flour', '200', 'g', 1);
		INSERT INTO recipe_steps (id, recipe_id, step_number, instruction) VALUES
			('88888888-8888-8888-8888-888888888888', '66666666-6666-6666-6666-666666666666', 1, 'Mix and cook');
		INSERT INTO meal_plans (id, name, start_date, end_date, is_published) VALUES
			('99999999-9999-9999-9999-999999999999', 'Week 1', '2026-01-12', '2026-01-18', true);
		INSERT INTO meal_plan_slots (id, meal_plan_id, date, meal_type, recipe_id, notes) VALUES
			('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '99999999-9999-9999-9999-999999999999', '2026-01-12', 'breakfast', '66666666-6666-6666-6666-666666666666', '');
		INSERT INTO meal_plan_shopping_list (id, meal_plan_id, ingredient_name, quantity_agg, unit, is_checked) VALUES
			('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '99999999-9999-9999-9999-999999999999', 'Flour', '200', 'g', false);
		INSERT INTO pomodoro_sessions (id, user_id, focus_minutes, elapsed_s, started_at, completed_at) VALUES
			('cccccccc-cccc-cccc-cccc-cccccccccccc', 'dev-user', 25, 1500, NOW(), NOW());
		INSERT INTO insight_reports (id, user_id, week_start, report_json, report_text) VALUES
			('dddddddd-dddd-dddd-dddd-dddddddddddd', 'dev-user', '2026-01-12', '{"score":5}', 'Good week');
	`
	if _, err := pool.Exec(ctx, seed); err != nil {
		t.Fatalf("seed data: %v", err)
	}
}

func TestBackupService_ExportImportRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupBackupTestPostgres(t)
	seedBackupTestData(t, ctx, pool)
	service := NewService(pool)

	// 1. Export a backup while data exists.
	data, err := service.Export(ctx, "test-version")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	var backup BackupFile
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("unmarshal exported backup: %v", err)
	}
	if backup.Version != BackupFormatVersion {
		t.Fatalf("version = %q, want %q", backup.Version, BackupFormatVersion)
	}
	if backup.AppVersion != "test-version" {
		t.Fatalf("app_version = %q, want test-version", backup.AppVersion)
	}
	expectOne(t, backup.RowCounts, "tasks")
	expectOne(t, backup.RowCounts, "task_lists")
	expectOne(t, backup.RowCounts, "task_completions")
	expectOne(t, backup.RowCounts, "planner_entries")
	expectOne(t, backup.RowCounts, "books")
	expectOne(t, backup.RowCounts, "recipes")
	expectOne(t, backup.RowCounts, "recipe_tags")
	expectOne(t, backup.RowCounts, "recipe_ingredients")
	expectOne(t, backup.RowCounts, "recipe_steps")
	expectOne(t, backup.RowCounts, "meal_plans")
	expectOne(t, backup.RowCounts, "meal_plan_slots")
	expectOne(t, backup.RowCounts, "meal_plan_shopping_list")
	expectOne(t, backup.RowCounts, "pomodoro_sessions")
	expectOne(t, backup.RowCounts, "insight_reports")
	if backup.RowCounts["feature_flags"] < 1 {
		t.Fatalf("feature_flags row count = %d, want >= 1 (seeded by migration)", backup.RowCounts["feature_flags"])
	}

	// 2. Simulate a lost database: wipe every backed-up table.
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+strings.Join(tableNames(), ", ")+" CASCADE"); err != nil {
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
	result, err := service.Import(ctx, data)
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

	var reportJSON map[string]any
	if err := pool.QueryRow(ctx,
		`SELECT report_json FROM insight_reports WHERE id = 'dddddddd-dddd-dddd-dddd-dddddddddddd'`,
	).Scan(&reportJSON); err != nil {
		t.Fatalf("query restored insight report: %v", err)
	}
	if reportJSON["score"] != float64(5) {
		t.Fatalf("restored report_json = %v, want score=5", reportJSON)
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

func TestBackupService_ImportRejectsBadInput(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}
	ctx, pool := setupBackupTestPostgres(t)
	service := NewService(pool)

	// Malformed JSON → ErrInvalidBackup, nothing touched.
	if _, err := service.Import(ctx, []byte(`{"tables": [`)); !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("malformed JSON: want ErrInvalidBackup, got %v", err)
	}

	// Unknown version → ErrUnsupportedVersion, nothing touched.
	bad := []byte(`{"version":"99","tables":{}}`)
	if _, err := service.Import(ctx, bad); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("bad version: want ErrUnsupportedVersion, got %v", err)
	}
}

func expectOne(t *testing.T, counts map[string]int, table string) {
	t.Helper()
	if got := counts[table]; got != 1 {
		t.Fatalf("%s row count = %d, want 1", table, got)
	}
}

func tableNames() []string {
	names := make([]string, len(backupTables))
	for i, t := range backupTables {
		names[i] = t.name
	}
	return names
}

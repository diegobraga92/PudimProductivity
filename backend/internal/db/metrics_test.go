package db

import "testing"

func TestOperationFor(t *testing.T) {
	cases := map[string]string{
		"SELECT * FROM tasks WHERE status = $1":             "select tasks",
		"SELECT * FROM task_completions WHERE task_id = $1": "select task_completions",
		"INSERT INTO tasks (id, title) VALUES ($1, $2)":     "insert tasks",
		"INSERT INTO task_completions ...":                  "insert task_completions",
		"UPDATE tasks SET status = $1 WHERE id = $2":        "update tasks",
		"DELETE FROM tasks WHERE id = $1":                   "delete tasks",
		"  SELECT\tDISTINCT id FROM tasks\nLIMIT 10":         "select tasks",
		"CREATE TABLE IF NOT EXISTS schema_migrations (":    "create schema_migrations",
		"CREATE INDEX IF NOT EXISTS idx_tasks_habits ON tasks": "create",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS x":      "alter tasks",
		"DROP TABLE IF EXISTS pgbench_accounts":             "drop pgbench_accounts",
		"":                                                  "unknown",
		"SELECT 1":                                          "select",
	}
	for sql, want := range cases {
		if got := operationFor(sql); got != want {
			t.Errorf("operationFor(%q) = %q, want %q", sql, got, want)
		}
	}
}

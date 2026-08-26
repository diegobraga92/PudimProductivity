package persistence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/tasklist"
)

// syncSchema compiles the SyncBundle schema from sync.yaml.
func syncSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "..", "api", "openapi", "sync-v1.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract %s: %v", path, err)
	}
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal contract %s: %v", path, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("sync-v1.yaml", doc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	schema, err := compiler.Compile("sync-v1.yaml#/components/schemas/SyncBundle")
	if err != nil {
		t.Fatalf("compile contract: %v", err)
	}
	return schema
}

// stubRepo returns a fixed ChangeSet so the handler's wire mapping can be
// exercised without a database.
type stubRepo struct{ cs *ChangeSet }

func (s stubRepo) Bundle(_ context.Context, _ time.Time) (*ChangeSet, error) {
	return s.cs, nil
}

// TestSyncResponseConformsToContract exercises the full sync endpoint and
// validates the JSON response against the SyncBundle schema in sync.yaml.
func TestSyncResponseConformsToContract(t *testing.T) {
	schema := syncSchema(t)

	now := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	cs := &ChangeSet{
		Timestamp: now,
		Tasks: []*task.Task{{
			ID:             "00000000-0000-0000-0000-000000000001",
			Title:          "Contract test",
			Status:         task.TaskStatusTodo,
			RecurrenceDays: []string{"mon", "fri"},
			CreatedAt:      now,
			UpdatedAt:      now,
		}},
		DeletedTaskIDs: []string{"00000000-0000-0000-0000-000000000002"},
		Completions: []*task.TaskCompletion{{
			ID:            "00000000-0000-0000-0000-000000000003",
			TaskID:        "00000000-0000-0000-0000-000000000001",
			CompletedDate: now,
			CreatedAt:     now,
		}},
		DeletedCompletionIDs: []string{"00000000-0000-0000-0000-000000000004"},
		TaskLists: []*tasklist.TaskList{{
			ID:          "00000000-0000-0000-0000-00000000000a",
			Name:        "Contract",
			Description: "offline sync contract",
			OwnerID:     "user-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		}},
		DeletedTaskListIDs: []string{"00000000-0000-0000-0000-00000000000b"},
		Shares: []*tasklist.Share{{
			ListID:     "00000000-0000-0000-0000-00000000000a",
			SharedWith: "user-2",
			Role:       tasklist.RoleEditor,
			CreatedAt:  now,
		}},
		DeletedShareKeys: []string{"00000000-0000-0000-0000-00000000000a:user-2"},
	}

	r := chi.NewRouter()
	RegisterSyncStoreRoutes(r, stubRepo{cs: cs})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	res, err := http.Get(srv.URL + "/api/v1/sync")
	if err != nil {
		t.Fatalf("get sync: %v", err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	if res.StatusCode != http.StatusOK {
		t.Fatalf("sync status = %d, want 200", res.StatusCode)
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read sync response: %v", err)
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		t.Fatalf("sync response not valid JSON: %v", err)
	}
	if err := schema.Validate(instance); err != nil {
		t.Errorf("sync response violates api/openapi/sync-v1.yaml: %v\nraw: %s", err, truncate(string(raw)))
	}
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

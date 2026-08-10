package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
)

// contractSchema compiles api/ws/events-v1.json — the source of truth for the
// WebSocket message format.
func contractSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	path := filepath.Join("..", "..", "..", "api", "ws", "events-v1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract %s: %v", path, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("events-v1.json", doc); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	schema, err := compiler.Compile("events-v1.json")
	if err != nil {
		t.Fatalf("compile contract: %v", err)
	}
	return schema
}

// stubResolver gives the contract-test connection membership of the lists used
// in scoped event payloads, so Phase 8 list-scoped events are delivered.
type stubResolver struct{ listIDs []string }

func (s stubResolver) ListIDsForUser(_ context.Context, _, _ string) ([]string, error) {
	return s.listIDs, nil
}

// TestWsEventsConformToContract publishes every task event type through the hub
// and validates the received JSON against api/ws/events-v1.json. This is the
// contract test that prevents spec drift between the Go event bus and the
// documented WebSocket schema.
func TestWsEventsConformToContract(t *testing.T) {
	schema := contractSchema(t)
	bus := eventbus.NewInMemoryBus()
	t.Cleanup(func() { _ = bus.Close() })
	hub := NewHub(bus, Config{ReplayBufferSize: 100})
	hub.SetMembershipResolver(stubResolver{
		listIDs: []string{"00000000-0000-0000-0000-00000000000a"},
	})
	if err := hub.Start(context.Background()); err != nil {
		t.Fatalf("hub start: %v", err)
	}
	t.Cleanup(hub.Close)

	srv := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv, 0), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "done") })

	payloads := []struct {
		typ     eventbus.EventType
		payload map[string]interface{}
	}{
		{eventbus.EventTaskCreated, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "title": "Contract test", "status": "todo",
			"recurrence_days": []string{"mon", "fri"}, "created_at": "2026-08-09T12:00:00Z", "updated_at": "2026-08-09T12:00:00Z",
		}},
		{eventbus.EventTaskUpdated, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "title": "Contract test", "status": "done",
			"created_at": "2026-08-09T12:00:00Z", "updated_at": "2026-08-09T12:01:00Z",
		}},
		{eventbus.EventTaskDeleted, map[string]interface{}{"id": "00000000-0000-0000-0000-000000000001"}},
		{eventbus.EventTaskCompleted, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000002", "task_id": "00000000-0000-0000-0000-000000000001",
			"title": "Contract test", "completed_date": "2026-08-09",
		}},
		{eventbus.EventTaskUncompleted, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "title": "Contract test", "completed_date": "2026-08-09",
		}},
		// Phase 8: collaboration events (list-scoped; the test connection is a
		// member of the list via stubResolver).
		{eventbus.EventTaskMerged, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "title": "Merged title", "status": "todo",
			"list_id":    "00000000-0000-0000-0000-00000000000a",
			"created_at": "2026-08-09T12:00:00Z", "updated_at": "2026-08-09T12:02:00Z",
		}},
		{eventbus.EventTaskListShared, map[string]interface{}{
			"list_id": "00000000-0000-0000-0000-00000000000a", "shared_with": "user-2",
			"role": "editor", "shared_by": "user-1",
		}},
		{eventbus.EventTaskListUnshared, map[string]interface{}{
			"list_id": "00000000-0000-0000-0000-00000000000a", "shared_with": "user-2",
			"removed_by": "user-1",
		}},
		{eventbus.EventPresenceOnline, map[string]interface{}{
			"user_id": "user-1", "list_ids": []string{"00000000-0000-0000-0000-00000000000a"},
		}},
		{eventbus.EventPresenceOffline, map[string]interface{}{
			"user_id": "user-1",
		}},
		// Phase 5: book tracking + meal planning.
		{eventbus.EventBookAdded, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "isbn": "9780553386790", "title": "Permanent Record",
		}},
		{eventbus.EventMealPlanPublished, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001",
		}},
		// Phase 5a: recipes.
		{eventbus.EventRecipeCreated, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "title": "Pancakes", "difficulty": "easy",
		}},
		{eventbus.EventRecipeUpdated, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001", "title": "Pancakes v2", "difficulty": "medium",
		}},
		{eventbus.EventRecipeDeleted, map[string]interface{}{
			"id": "00000000-0000-0000-0000-000000000001",
		}},
	}

	for i, p := range payloads {
		if err := bus.Publish(ctx, p.typ, p.payload); err != nil {
			t.Fatalf("publish %s: %v", p.typ, err)
		}

		_, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read event %d (%s): %v", i, p.typ, err)
		}

		// Validate the raw event JSON against the contract schema.
		var instance any
		if err := json.Unmarshal(data, &instance); err != nil {
			t.Fatalf("event %d not valid JSON: %v", i, err)
		}
		if err := schema.Validate(instance); err != nil {
			t.Errorf("event %d (%s) violates api/ws/events-v1.json: %v\nraw: %s",
				i, p.typ, err, truncate(string(data)))
		}
	}
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

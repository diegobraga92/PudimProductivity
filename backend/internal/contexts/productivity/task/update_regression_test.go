package task

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestService_UpdateTask_OneOffTitleRenameWithEmptyRecurrenceDays(t *testing.T) {
	task, err := NewTask("t-1", "Ergométrico", nil)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}

	svc := NewTaskService(newFakeRepo(task), nil, nil)

	title := "Ergométrico renomeado"
	empty := []string{}

	updated, err := svc.UpdateTask(context.Background(), "t-1", &title, nil, &empty, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	if updated.Title != title {
		t.Errorf("Title: got %q, want %q", updated.Title, title)
	}

	if updated.IsHabit() {
		t.Error("expected task to remain a one-off")
	}
}

func TestHandler_UpdateTask_WebEditorPayload(t *testing.T) {
	task, err := NewTask("t-1", "Ergométrico", nil)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}

	handler := NewHandler(NewTaskService(newFakeRepo(task), nil, nil))

	body := `{"title":"Ergométrico renomeado","recurrence_days":[],"start_time":null,"end_time":null,"color":null,"scheduled_date":null,"alarm_minutes":null}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/t-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", "t-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	handler.UpdateTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
}

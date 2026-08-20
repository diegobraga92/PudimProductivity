// Package audit provides structured audit logging for events.
package audit

import (
	"encoding/json"
	"time"
)

// TODO: Check for completeness

// Action constants for audit log entries.
// Follows the pattern "<resource>.<verb>".
const (
	ActionTaskCreated     = "task.created"
	ActionTaskUpdated     = "task.updated"
	ActionTaskDeleted     = "task.deleted"
	ActionTaskCompleted   = "task.completed"
	ActionTaskUncompleted = "task.uncompleted"

	ActionListCreated = "list.created"
	ActionListUpdated = "list.updated"
	ActionListDeleted = "list.deleted"

	ActionFeatureToggled = "feature.toggled"

	ActionFocusStarted   = "focus.started"
	ActionFocusCompleted = "focus.completed"

	ActionLibraryItemAdded     = "library.item.added"
	ActionLibraryItemUpdated   = "library.item.updated"
	ActionLibraryItemDeleted   = "library.item.deleted"
	ActionLibraryItemsImported = "library.items.imported"

	ActionRecipeCreated = "recipe.created"
	ActionRecipeUpdated = "recipe.updated"
	ActionRecipeDeleted = "recipe.deleted"

	ActionScoreProviderUpdated = "score_provider.updated"
)

// Resource constants to identify event domain.
const (
	ResourceTasks          = "tasks"
	ResourceTaskLists      = "task_lists"
	ResourceFeatures       = "feature_flags"
	ResourcePomodoro       = "pomodoro"
	ResourceLibraryItems   = "library_items"
	ResourceRecipes        = "recipes"
	ResourceScoreProviders = "score_providers"
)

// Entry represents a single audit log record.
type Entry struct {
	ID         string          `json:"id"`
	ActorID    string          `json:"actor_id"`
	Action     string          `json:"action"`
	Resource   string          `json:"resource"`
	ResourceID string          `json:"resource_id,omitempty"`
	OldValues  json.RawMessage `json:"old_values,omitempty"`
	NewValues  json.RawMessage `json:"new_values,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

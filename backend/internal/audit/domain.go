package audit

import (
	"encoding/json"
	"time"
)

// Action constants for audit log entries.
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

	// Library media tracking (replaces Phase 5 book tracking).
	ActionLibraryItemAdded     = "library.item.added"
	ActionLibraryItemUpdated   = "library.item.updated"
	ActionLibraryItemDeleted   = "library.item.deleted"
	ActionLibraryItemsImported = "library.items.imported"

	// Phase 5a: recipes.
	ActionRecipeCreated = "recipe.created"
	ActionRecipeUpdated = "recipe.updated"
	ActionRecipeDeleted = "recipe.deleted"

	// Runtime score-provider configuration (admin UI).
	ActionScoreProviderUpdated = "score_provider.updated"
)

// Resource constants.
const (
	ResourceTasks     = "tasks"
	ResourceTaskLists = "task_lists"
	ResourceFeatures  = "feature_flags"
	ResourcePomodoro  = "pomodoro"

	// Library media tracking (replaces Phase 5 book tracking).
	ResourceLibraryItems = "library_items"

	// Phase 5a: recipes.
	ResourceRecipes = "recipes"

	// Runtime score-provider configuration (admin UI).
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

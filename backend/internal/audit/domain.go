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

	// Phase 5: book tracking + meal planning.
	ActionBookAdded       = "book.added"
	ActionBookUpdated     = "book.updated"
	ActionBookDeleted     = "book.deleted"
	ActionMealPlanCreated = "mealplan.created"
	ActionMealPlanUpdated = "mealplan.updated"
	ActionMealPlanDeleted = "mealplan.deleted"

	// Phase 5a: recipes.
	ActionRecipeCreated = "recipe.created"
	ActionRecipeUpdated = "recipe.updated"
	ActionRecipeDeleted = "recipe.deleted"
)

// Resource constants.
const (
	ResourceTasks     = "tasks"
	ResourceTaskLists = "task_lists"
	ResourceFeatures  = "feature_flags"
	ResourcePomodoro  = "pomodoro"

	// Phase 5: book tracking + meal planning.
	ResourceBooks     = "books"
	ResourceMealPlans = "meal_plans"

	// Phase 5a: recipes.
	ResourceRecipes = "recipes"
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

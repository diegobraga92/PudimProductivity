package notification

import (
	"fmt"

	"github.com/diegobraga92/pudimproductivity/backend/internal/platform/eventbus"
)

// renderNotification builds the push/email title and body for a task event.
// The payload arrives from RabbitMQ as a JSON object (map[string]interface{}).
// Returns ok=false for event types that should not produce notifications.
func renderNotification(event eventbus.Event) (title, body string, ok bool) {
	payload, _ := event.Payload.(map[string]interface{})

	switch event.Type {
	case eventbus.EventTaskCreated:
		return "New task", fmt.Sprintf("Task “%s” was created.", field(payload, "title")), true
	case eventbus.EventTaskUpdated:
		return "Task updated", fmt.Sprintf("Task “%s” was updated.", field(payload, "title")), true
	case eventbus.EventTaskDeleted:
		return "Task deleted", "A task was removed from your list.", true
	case eventbus.EventTaskCompleted:
		return "Habit completed 🎉", fmt.Sprintf("“%s” done for %s.", field(payload, "title"), field(payload, "completed_date")), true
	case eventbus.EventTaskUncompleted:
		return "Habit uncompleted", fmt.Sprintf("The completion for “%s” was removed.", field(payload, "title")), true
	default:
		return "", "", false
	}
}

// field reads a string field from the JSON payload map.
func field(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// extractTaskID pulls the task id out of an event payload for audit purposes.
func extractTaskID(event eventbus.Event) string {
	payload, _ := event.Payload.(map[string]interface{})
	if id := field(payload, "task_id"); id != "" {
		return id
	}
	return field(payload, "id")
}

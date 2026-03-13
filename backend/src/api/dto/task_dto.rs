use crate::domain::task::Task as DomainTask;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskDto {
    pub id: String,
    pub list_id: String,
    pub title: String,                // Maps to 'text' in database
    pub completed: bool,              // Maps to 'done' in database
    pub order: i32,                   // Not in schema, default to 0
    pub due_date: Option<String>,     // Not in schema
    pub recurrence: Option<String>,   // Maps to 'repeat_on' in database
    pub streak_count: Option<i32>,    // Not in schema
    pub completed_at: Option<String>, // From task_completions table
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateTaskRequest {
    pub list_id: String,
    pub title: String,
    pub completed: bool,
    pub order: i32,
    pub due_date: Option<String>,
    pub recurrence: Option<String>,
    pub streak_count: Option<i32>,
    pub completed_at: Option<String>,
}

impl TaskDto {
    pub fn from_domain(task: DomainTask, completed_at: Option<DateTime<Utc>>) -> Self {
        // Convert repeat_on array to recurrence string
        let recurrence = task.repeat_on.as_ref().map(|repeat_on| {
            // Convert array of weekday numbers to string representation
            // This is a simplified conversion
            format!("{:?}", repeat_on)
        });

        TaskDto {
            id: task.id.to_string(),
            list_id: task.list_id.to_string(),
            title: task.text,     // Map text to title
            completed: task.done, // Map done to completed
            order: 0,             // Default order since not in schema
            due_date: None,       // Not in schema
            recurrence,
            streak_count: None, // Not in schema
            completed_at: completed_at.map(|date| date.to_rfc3339()),
            created_at: task
                .created_at
                .map(|date| date.to_rfc3339())
                .unwrap_or_default(),
            updated_at: "".to_string(), // Not in domain model yet
        }
    }
}

// TODO REVIEW

impl CreateTaskRequest {
    pub fn to_domain(&self) -> Result<DomainTask, String> {
        let id = Uuid::new_v4();
        let list_id = Uuid::parse_str(&self.list_id)
            .map_err(|_| format!("Invalid list ID format: {}", self.list_id))?;

        // Parse completed_at if provided
        let completed_at = self
            .completed_at
            .as_ref()
            .and_then(|date_str| DateTime::parse_from_rfc3339(date_str).ok())
            .map(|date| date.with_timezone(&Utc));

        // Convert recurrence string to repeat_on array (simplified)
        let repeat_on = self.recurrence.as_ref().map(|_| {
            // This is a simplified conversion - in a real app you'd parse the recurrence string
            // For now, return an empty vector
            Vec::new()
        });

        Ok(DomainTask {
            id,
            list_id,
            text: self.title.clone(),
            done: self.completed,
            repeat_on,
            created_at: completed_at.or_else(|| Some(Utc::now())),
            deleted_at: None,
        })
    }
}

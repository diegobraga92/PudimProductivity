use crate::domain::task::{Task, TaskCompletion};
use chrono::{DateTime, Utc, Weekday};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskRequest {
    pub list_id: Uuid,
    pub text: String,
    pub done: bool,
    pub repeat_on: Option<Vec<Weekday>>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskResponse {
    pub id: Uuid,
    pub list_id: Uuid,
    pub text: String,
    pub done: bool,
    pub repeat_on: Option<Vec<Weekday>>,
    pub created_at: Option<DateTime<Utc>>,
    pub deleted_at: Option<DateTime<Utc>>,
}

impl From<Task> for TaskResponse {
    fn from(task: Task) -> Self {
        Self {
            id: task.id,
            list_id: task.list_id,
            text: task.text,
            done: task.done,
            repeat_on: task.repeat_on,
            created_at: task.created_at,
            deleted_at: task.deleted_at,
        }
    }
}

// TODO REVIEW

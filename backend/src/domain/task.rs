use chrono::{DateTime, Utc, Weekday};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct Task {
    pub id: Uuid,
    pub list_id: Uuid,
    pub text: String,
    pub done: bool,
    pub repeat_on: Option<Vec<Weekday>>,
    pub created_at: Option<DateTime<Utc>>,
    pub deleted_at: Option<DateTime<Utc>>,
}

pub struct TaskCompletion {
    pub task_id: Uuid,
    pub date: DateTime<Utc>,
}

// TODO REVIEW

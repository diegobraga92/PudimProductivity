use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Serialize, Deserialize, Debug)]
pub enum TaskType {
    Regular = 0,
    Daily = 1,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Task {
    pub id: Uuid,
    pub list_id: Uuid,
    pub title: String,
    pub task_type: TaskType,
    pub checked: bool,
}

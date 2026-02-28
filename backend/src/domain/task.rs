use chrono::Weekday;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
pub struct Task {
    pub id: Uuid,
    pub list_id: Uuid,
    pub text: String,
    pub done: bool,
    pub repeat_on: Option<Vec<Weekday>>,
}

// TODO

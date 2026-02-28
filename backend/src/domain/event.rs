use chrono::Utc;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Event {
    pub id: Uuid,
    pub entity_id: Uuid,
    pub entity_type: String,
    pub event_type: String,
    pub payload: serde_json::Value,
    pub timestamp: i64,
    pub device_id: Uuid,
}

impl Event {
    pub fn new(
        entity_id: Uuid,
        entity_type: String,
        event_type: String,
        payload: serde_json::Value,
        device_id: Uuid,
    ) -> Self {
        Self {
            id: Uuid::new_v4(),
            entity_id,
            entity_type,
            event_type,
            payload,
            timestamp: Utc::now().timestamp_millis(),
            device_id,
        }
    }
}

// TODO

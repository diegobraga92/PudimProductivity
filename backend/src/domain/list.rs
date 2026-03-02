use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Serialize, Deserialize)]
enum ListType {
    Todo = 0,
    Daily = 1,
    Collection = 2,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct List {
    pub id: Uuid,
    pub parent_id: Option<Uuid>,
    pub name: String,
    pub list_type: ListType,
}

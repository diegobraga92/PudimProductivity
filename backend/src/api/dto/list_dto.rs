use crate::domain::list::{List, ListType};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ListRequest {
    pub parent_id: Option<Uuid>,
    pub name: String,

    #[serde(rename = "type")]
    pub list_type: ListType,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ListResponse {
    pub id: Uuid,
    pub parent_id: Option<Uuid>,
    pub name: String,

    #[serde(rename = "type")]
    pub list_type: ListType,
}

impl From<List> for ListResponse {
    fn from(list: List) -> Self {
        Self {
            id: list.id,
            parent_id: list.parent_id,
            name: list.name,
            list_type: list.list_type,
        }
    }
}

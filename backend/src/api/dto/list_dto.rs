use crate::domain::list::{List as DomainList, ListType};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ListDto {
    pub id: String,
    pub parent_id: Option<String>,
    pub name: String,
    pub r#type: String, // "todo", "daily", or "collection"
    pub config: Value,  // Empty object for now
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateListRequest {
    pub parent_id: Option<String>,
    pub name: String,
    pub r#type: String, // "todo", "daily", or "collection"
    pub config: Value,
}

impl ListDto {
    pub fn from_domain(list: DomainList) -> Self {
        let type_str = match list.list_type {
            ListType::Todo => "todo".to_string(),
            ListType::Daily => "daily".to_string(),
            ListType::Collection => "collection".to_string(),
        };

        ListDto {
            id: list.id.to_string(),
            parent_id: list.parent_id.map(|id| id.to_string()),
            name: list.name,
            r#type: type_str,
            config: Value::Object(Default::default()), // Empty config for now
            created_at: "".to_string(),                // Not in domain model yet
            updated_at: "".to_string(),                // Not in domain model yet
        }
    }
}

// TODO REVIEW

impl CreateListRequest {
    pub fn to_domain(&self) -> Result<DomainList, String> {
        let id = Uuid::new_v4();
        let list_type = match self.r#type.as_str() {
            "todo" => ListType::Todo,
            "daily" => ListType::Daily,
            "collection" => ListType::Collection,
            _ => return Err(format!("Invalid list type: {}", self.r#type)),
        };

        let parent_id = self
            .parent_id
            .as_ref()
            .and_then(|pid| Uuid::parse_str(pid).ok());

        Ok(DomainList {
            id,
            parent_id,
            name: self.name.clone(),
            list_type,
        })
    }
}

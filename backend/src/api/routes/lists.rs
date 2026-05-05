use crate::api::dto::list_dto::{CreateListRequest, ListDto};
use crate::api::routes::AppState;
use axum::{
    extract::{Path, State},
    http::StatusCode,
    Json,
};
use std::sync::Arc;
use uuid::Uuid;

#[derive(Serialize)]
pub struct ApiResponse<T> {
    data: T,
    success: bool,
    message: Option<String>,
}

#[derive(Serialize)]
pub struct ErrorResponse {
    message: String,
}

fn ok<T>(data: T) -> Json<ApiResponse<T>> {
    Json(ApiResponse {
        data,
        success: true,
        message: None,
    })
}

pub async fn get_lists(
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<Vec<ListDto>>>, (StatusCode, Json<ErrorResponse>)> {
    let lists = state
        .list_repository
        .get_all()
        .map_err(|err| internal_error(err))?;

    let list_dtos = lists.into_iter().map(ListDto::from_domain).collect();
    Ok(ok(list_dtos))
}

pub async fn create_list(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateListRequest>,
) -> Result<Json<ApiResponse<ListDto>>, (StatusCode, Json<ErrorResponse>)> {
    let domain_list = request.to_domain().map_err(|err| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse { message: err }),
        )
    })?;

    let created_list = state
        .list_repository
        .create(domain_list)
        .map_err(|err| internal_error(err))?;

    Ok(ok(ListDto::from_domain(created_list)))
}

pub async fn get_tasks_for_list(
    State(state): State<Arc<AppState>>,
    Path(list_id): Path<String>,
) -> Result<
    Json<ApiResponse<Vec<crate::api::dto::task_dto::TaskDto>>>,
    (StatusCode, Json<ErrorResponse>),
> {
    let list_uuid = Uuid::parse_str(&list_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid list ID format".to_string(),
            }),
        )
    })?;

    // Get tasks for this list
    let tasks = state
        .task_repository
        .get_by_list_id(list_uuid)
        .map_err(|err| internal_error(err))?;

    // Convert to DTOs
    use crate::api::dto::task_dto::TaskDto;
    let task_dtos = tasks
        .into_iter()
        .map(|task| TaskDto::from_domain(task, None)) // TODO: Get completion dates
        .collect();

    Ok(ok(task_dtos))
}

fn internal_error<E: std::fmt::Display>(error: E) -> (StatusCode, Json<ErrorResponse>) {
    (
        StatusCode::INTERNAL_SERVER_ERROR,
        Json(ErrorResponse {
            message: format!("Internal server error: {error}"),
        }),
    )
}

use serde::Serialize;

// TODO REVIEW

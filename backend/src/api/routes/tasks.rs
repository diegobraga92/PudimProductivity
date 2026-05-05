use crate::api::dto::task_dto::{CreateTaskRequest, TaskDto};
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

pub async fn create_task(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateTaskRequest>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    let domain_task = request.to_domain().map_err(|err| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse { message: err }),
        )
    })?;

    let created_task = state
        .task_repository
        .create(domain_task)
        .map_err(|err| internal_error(err))?;

    // For now, return empty completed_at since we don't have completion tracking in DTO conversion
    Ok(ok(TaskDto::from_domain(created_task, None)))
}

pub async fn complete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    update_task_completion(state, task_id, true).await
}

pub async fn reopen_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    update_task_completion(state, task_id, false).await
}

async fn update_task_completion(
    state: Arc<AppState>,
    task_id: String,
    completed: bool,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    let task_uuid = Uuid::parse_str(&task_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid task ID format".to_string(),
            }),
        )
    })?;

    let updated_task = state
        .task_repository
        .update_completion(task_uuid, completed)
        .map_err(|err| match err {
            diesel::result::Error::NotFound => (
                StatusCode::NOT_FOUND,
                Json(ErrorResponse {
                    message: "Task not found".to_string(),
                }),
            ),
            _ => internal_error(err),
        })?;

    // For now, return empty completed_at since we don't have completion tracking in DTO conversion
    Ok(ok(TaskDto::from_domain(updated_task, None)))
}

pub async fn delete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<()>>, (StatusCode, Json<ErrorResponse>)> {
    let task_uuid = Uuid::parse_str(&task_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid task ID format".to_string(),
            }),
        )
    })?;

    state
        .task_repository
        .soft_delete(task_uuid)
        .map_err(|err| match err {
            diesel::result::Error::NotFound => (
                StatusCode::NOT_FOUND,
                Json(ErrorResponse {
                    message: "Task not found".to_string(),
                }),
            ),
            _ => internal_error(err),
        })?;

    Ok(ok(()))
}

fn internal_error<E: std::fmt::Display>(error: E) -> (StatusCode, Json<ErrorResponse>) {
    (
        StatusCode::INTERNAL_SERVER_ERROR,
        Json(ErrorResponse {
            message: format!("Internal server error: {error}"),
        }),
    )
}

use diesel;
use serde::Serialize;

// TODO REVIEW

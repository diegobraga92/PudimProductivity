use crate::infrastructure::schema::{lists, tasks};
use axum::{
    extract::{Path, State},
    http::StatusCode,
    routing::{delete, get, patch, post},
    Json, Router,
};
use chrono::{DateTime, Utc};
use diesel::{
    prelude::*,
    r2d2::{ConnectionManager, Pool},
    Insertable, PgConnection, Queryable, Selectable,
};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::sync::Arc;
use tower_http::cors::{Any, CorsLayer};
use uuid::Uuid;

pub type DbPool = Pool<ConnectionManager<PgConnection>>;

#[derive(Clone)]
pub struct AppState {
    pub pool: DbPool,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ApiResponse<T> {
    data: T,
    success: bool,
    message: Option<String>,
}

#[derive(Serialize)]
pub struct ErrorResponse {
    message: String,
}

#[derive(Queryable, Selectable, Serialize)]
#[diesel(table_name = lists)]
#[diesel(check_for_backend(diesel::pg::Pg))]
#[serde(rename_all = "camelCase")]
pub struct List {
    pub id: String,
    pub user_id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub type_: String,
    pub config: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Insertable)]
#[diesel(table_name = lists)]
pub struct NewList<'a> {
    pub id: &'a str,
    pub user_id: &'a str,
    pub name: &'a str,
    pub type_: &'a str,
    pub config: &'a str,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Queryable, Selectable, Serialize)]
#[diesel(table_name = tasks)]
#[diesel(check_for_backend(diesel::pg::Pg))]
#[serde(rename_all = "camelCase")]
pub struct Task {
    pub id: String,
    pub list_id: String,
    pub title: String,
    pub completed: bool,
    pub order_index: i32,
    pub due_date: Option<DateTime<Utc>>,
    pub recurrence: Option<String>,
    pub streak_count: Option<i32>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Insertable)]
#[diesel(table_name = tasks)]
pub struct NewTask<'a> {
    pub id: &'a str,
    pub list_id: &'a str,
    pub title: &'a str,
    pub completed: bool,
    pub order_index: i32,
    pub due_date: Option<DateTime<Utc>>,
    pub recurrence: Option<&'a str>,
    pub streak_count: Option<i32>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ListDto {
    id: String,
    user_id: String,
    name: String,
    r#type: String,
    config: Value,
    created_at: String,
    updated_at: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateListRequest {
    user_id: String,
    name: String,
    r#type: String,
    config: Value,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskDto {
    id: String,
    list_id: String,
    title: String,
    completed: bool,
    order: i32,
    due_date: Option<String>,
    recurrence: Option<String>,
    streak_count: Option<i32>,
    completed_at: Option<String>,
    created_at: String,
    updated_at: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateTaskRequest {
    list_id: String,
    title: String,
    completed: bool,
    order: i32,
    due_date: Option<String>,
    recurrence: Option<String>,
    streak_count: Option<i32>,
    completed_at: Option<String>,
}

fn ok<T>(data: T) -> Json<ApiResponse<T>> {
    Json(ApiResponse {
        data,
        success: true,
        message: None,
    })
}

pub fn create_router(state: AppState) -> Router {
    Router::new()
        .route("/lists", get(get_lists).post(create_list))
        .route("/lists/:list_id/tasks", get(get_tasks_for_list))
        .route("/tasks", post(create_task))
        .route("/tasks/:task_id", delete(delete_task))
        .route("/tasks/:task_id/complete", patch(complete_task))
        .route("/tasks/:task_id/reopen", patch(reopen_task))
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_methods(Any)
                .allow_headers(Any),
        )
        .with_state(Arc::new(state))
}

async fn get_lists(
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<Vec<ListDto>>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let results: Vec<List> = lists::table
        .order_by(lists::created_at.asc())
        .load(&mut conn)
        .map_err(internal_error)?;

    let lists = results
        .into_iter()
        .map(|list| ListDto {
            id: list.id,
            user_id: list.user_id,
            name: list.name,
            r#type: list.type_,
            config: serde_json::from_str(&list.config).unwrap_or(Value::Object(Default::default())),
            created_at: list.created_at.to_rfc3339(),
            updated_at: list.updated_at.to_rfc3339(),
        })
        .collect();

    Ok(ok(lists))
}

async fn create_list(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateListRequest>,
) -> Result<Json<ApiResponse<ListDto>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let id = Uuid::new_v4().to_string();
    let created_at = Utc::now();
    let updated_at = created_at;
    let config_str = serde_json::to_string(&request.config).map_err(internal_error)?;

    let new_list = NewList {
        id: &id,
        user_id: &request.user_id,
        name: &request.name,
        type_: &request.r#type,
        config: &config_str,
        created_at,
        updated_at,
    };

    diesel::insert_into(lists::table)
        .values(&new_list)
        .execute(&mut conn)
        .map_err(internal_error)?;

    Ok(ok(ListDto {
        id,
        user_id: request.user_id,
        name: request.name,
        r#type: request.r#type,
        config: request.config,
        created_at: created_at.to_rfc3339(),
        updated_at: updated_at.to_rfc3339(),
    }))
}

async fn get_tasks_for_list(
    State(state): State<Arc<AppState>>,
    Path(list_id): Path<String>,
) -> Result<Json<ApiResponse<Vec<TaskDto>>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let results: Vec<Task> = tasks::table
        .filter(tasks::list_id.eq(&list_id))
        .order_by((tasks::order_index.asc(), tasks::created_at.asc()))
        .load(&mut conn)
        .map_err(internal_error)?;

    let tasks = results
        .into_iter()
        .map(|task| TaskDto {
            id: task.id,
            list_id: task.list_id,
            title: task.title,
            completed: task.completed,
            order: task.order_index,
            due_date: task.due_date.map(|d| d.to_rfc3339()),
            recurrence: task.recurrence,
            streak_count: task.streak_count,
            completed_at: task.completed_at.map(|c| c.to_rfc3339()),
            created_at: task.created_at.to_rfc3339(),
            updated_at: task.updated_at.to_rfc3339(),
        })
        .collect();

    Ok(ok(tasks))
}

async fn create_task(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateTaskRequest>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let id = Uuid::new_v4().to_string();
    let created_at = Utc::now();
    let updated_at = created_at;

    let due_date = request.due_date.as_ref().and_then(|d| {
        DateTime::parse_from_rfc3339(d)
            .ok()
            .map(|dt| dt.with_timezone(&Utc))
    });
    let completed_at = request.completed_at.as_ref().and_then(|c| {
        DateTime::parse_from_rfc3339(c)
            .ok()
            .map(|dt| dt.with_timezone(&Utc))
    });

    let new_task = NewTask {
        id: &id,
        list_id: &request.list_id,
        title: &request.title,
        completed: request.completed,
        order_index: request.order,
        due_date,
        recurrence: request.recurrence.as_deref(),
        streak_count: request.streak_count,
        completed_at,
        created_at,
        updated_at,
    };

    diesel::insert_into(tasks::table)
        .values(&new_task)
        .execute(&mut conn)
        .map_err(internal_error)?;

    Ok(ok(TaskDto {
        id,
        list_id: request.list_id,
        title: request.title,
        completed: request.completed,
        order: request.order,
        due_date: request.due_date,
        recurrence: request.recurrence,
        streak_count: request.streak_count,
        completed_at: request.completed_at,
        created_at: created_at.to_rfc3339(),
        updated_at: updated_at.to_rfc3339(),
    }))
}

async fn complete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    update_task_completion(state, task_id, true).await
}

async fn reopen_task(
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
    let mut conn = state.pool.get().map_err(internal_error)?;

    let updated_at = Utc::now();
    let completed_at = if completed { Some(Utc::now()) } else { None };

    let num_updated = diesel::update(tasks::table.filter(tasks::id.eq(&task_id)))
        .set((
            tasks::completed.eq(completed),
            tasks::completed_at.eq(completed_at),
            tasks::updated_at.eq(updated_at),
        ))
        .execute(&mut conn)
        .map_err(internal_error)?;

    if num_updated == 0 {
        return Err((
            StatusCode::NOT_FOUND,
            Json(ErrorResponse {
                message: "Task not found".to_string(),
            }),
        ));
    }

    let task: Task = tasks::table
        .filter(tasks::id.eq(&task_id))
        .first(&mut conn)
        .map_err(internal_error)?;

    Ok(ok(TaskDto {
        id: task.id,
        list_id: task.list_id,
        title: task.title,
        completed: task.completed,
        order: task.order_index,
        due_date: task.due_date.map(|d| d.to_rfc3339()),
        recurrence: task.recurrence,
        streak_count: task.streak_count,
        completed_at: task.completed_at.map(|c| c.to_rfc3339()),
        created_at: task.created_at.to_rfc3339(),
        updated_at: task.updated_at.to_rfc3339(),
    }))
}

async fn delete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<()>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let num_deleted = diesel::delete(tasks::table.filter(tasks::id.eq(&task_id)))
        .execute(&mut conn)
        .map_err(internal_error)?;

    if num_deleted == 0 {
        return Err((
            StatusCode::NOT_FOUND,
            Json(ErrorResponse {
                message: "Task not found".to_string(),
            }),
        ));
    }

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
// TODO

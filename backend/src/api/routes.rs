use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    routing::{delete, get, patch, post},
    Json, Router,
};
use chrono::Utc;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use sqlx::SqlitePool;
use std::sync::Arc;
use tower_http::cors::{Any, CorsLayer};
use uuid::Uuid;

use crate::application::sync::{get_events_since, insert_event};
use crate::domain::event::Event;

#[derive(Clone)]
pub struct AppState {
    pub pool: SqlitePool,
}

#[derive(Deserialize)]
pub struct SinceQuery {
    since: i64,
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

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UserDto {
    id: String,
    email: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct LoginData {
    token: String,
    user: UserDto,
}

#[derive(Deserialize)]
pub struct LoginRequest {
    email: String,
    password: String,
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
    order: i64,
    due_date: Option<String>,
    recurrence: Option<String>,
    streak_count: Option<i64>,
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
    order: i64,
    due_date: Option<String>,
    recurrence: Option<String>,
    streak_count: Option<i64>,
    completed_at: Option<String>,
}

fn ok<T>(data: T) -> Json<ApiResponse<T>> {
    Json(ApiResponse {
        data,
        success: true,
        message: None,
    })
}

fn now_iso() -> String {
    Utc::now().to_rfc3339()
}

pub fn create_router(state: AppState) -> Router {
    Router::new()
        .route("/auth/login", post(login))
        .route("/lists", get(get_lists).post(create_list))
        .route("/lists/:list_id/tasks", get(get_tasks_for_list))
        .route("/tasks", post(create_task))
        .route("/tasks/:task_id", delete(delete_task))
        .route("/tasks/:task_id/complete", patch(complete_task))
        .route("/tasks/:task_id/reopen", patch(reopen_task))
        .route("/events", post(post_event))
        .route("/events", get(get_events))
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_methods(Any)
                .allow_headers(Any),
        )
        .with_state(Arc::new(state))
}

async fn post_event(State(state): State<Arc<AppState>>, Json(event): Json<Event>) {
    insert_event(&state.pool, &event).await;
}

async fn get_events(
    State(state): State<Arc<AppState>>,
    Query(query): Query<SinceQuery>,
) -> Json<Vec<Event>> {
    let events = get_events_since(&state.pool, query.since).await;
    Json(events)
}

async fn login(
    State(state): State<Arc<AppState>>,
    Json(request): Json<LoginRequest>,
) -> Result<Json<ApiResponse<LoginData>>, (StatusCode, Json<ErrorResponse>)> {
    let row = sqlx::query_as::<_, (String, String, String)>(
        r#"
        SELECT id, email, password
        FROM users
        WHERE email = ?
        LIMIT 1
        "#,
    )
    .bind(&request.email)
    .fetch_optional(&state.pool)
    .await
    .map_err(internal_error)?;

    let Some((id, email, password)) = row else {
        return Err((
            StatusCode::UNAUTHORIZED,
            Json(ErrorResponse {
                message: "Invalid credentials".to_string(),
            }),
        ));
    };

    if request.password != password {
        return Err((
            StatusCode::UNAUTHORIZED,
            Json(ErrorResponse {
                message: "Invalid credentials".to_string(),
            }),
        ));
    }

    Ok(ok(LoginData {
        token: format!("dev-token-{id}"),
        user: UserDto { id, email },
    }))
}

async fn get_lists(
    State(state): State<Arc<AppState>>,
) -> Result<Json<ApiResponse<Vec<ListDto>>>, (StatusCode, Json<ErrorResponse>)> {
    let rows = sqlx::query_as::<_, (String, String, String, String, String, String, String)>(
        r#"
        SELECT id, user_id, name, type, config, created_at, updated_at
        FROM lists
        ORDER BY created_at ASC
        "#,
    )
    .fetch_all(&state.pool)
    .await
    .map_err(internal_error)?;

    let lists = rows
        .into_iter()
        .map(
            |(id, user_id, name, list_type, config, created_at, updated_at)| ListDto {
                id,
                user_id,
                name,
                r#type: list_type,
                config: serde_json::from_str(&config).unwrap_or(Value::Object(Default::default())),
                created_at,
                updated_at,
            },
        )
        .collect();

    Ok(ok(lists))
}

async fn create_list(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateListRequest>,
) -> Result<Json<ApiResponse<ListDto>>, (StatusCode, Json<ErrorResponse>)> {
    let id = Uuid::new_v4().to_string();
    let created_at = now_iso();
    let updated_at = created_at.clone();
    let config_str = serde_json::to_string(&request.config).map_err(internal_error)?;

    sqlx::query(
        r#"
        INSERT INTO lists (id, user_id, name, type, config, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        "#,
    )
    .bind(&id)
    .bind(&request.user_id)
    .bind(&request.name)
    .bind(&request.r#type)
    .bind(&config_str)
    .bind(&created_at)
    .bind(&updated_at)
    .execute(&state.pool)
    .await
    .map_err(internal_error)?;

    Ok(ok(ListDto {
        id,
        user_id: request.user_id,
        name: request.name,
        r#type: request.r#type,
        config: request.config,
        created_at,
        updated_at,
    }))
}

async fn get_tasks_for_list(
    State(state): State<Arc<AppState>>,
    Path(list_id): Path<String>,
) -> Result<Json<ApiResponse<Vec<TaskDto>>>, (StatusCode, Json<ErrorResponse>)> {
    let rows = sqlx::query_as::<
        _,
        (
            String,
            String,
            String,
            i64,
            i64,
            Option<String>,
            Option<String>,
            Option<i64>,
            Option<String>,
            String,
            String,
        ),
    >(
        r#"
        SELECT id, list_id, title, completed, order_index, due_date, recurrence, streak_count, completed_at, created_at, updated_at
        FROM tasks
        WHERE list_id = ?
        ORDER BY order_index ASC, created_at ASC
        "#,
    )
    .bind(&list_id)
    .fetch_all(&state.pool)
    .await
    .map_err(internal_error)?;

    let tasks = rows
        .into_iter()
        .map(
            |(
                id,
                list_id,
                title,
                completed,
                order,
                due_date,
                recurrence,
                streak_count,
                completed_at,
                created_at,
                updated_at,
            )| TaskDto {
                id,
                list_id,
                title,
                completed: completed != 0,
                order,
                due_date,
                recurrence,
                streak_count,
                completed_at,
                created_at,
                updated_at,
            },
        )
        .collect();

    Ok(ok(tasks))
}

async fn create_task(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateTaskRequest>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    let id = Uuid::new_v4().to_string();
    let created_at = now_iso();
    let updated_at = created_at.clone();

    sqlx::query(
        r#"
        INSERT INTO tasks
            (id, list_id, title, completed, order_index, due_date, recurrence, streak_count, completed_at, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        "#,
    )
    .bind(&id)
    .bind(&request.list_id)
    .bind(&request.title)
    .bind(if request.completed { 1 } else { 0 })
    .bind(request.order)
    .bind(&request.due_date)
    .bind(&request.recurrence)
    .bind(request.streak_count)
    .bind(&request.completed_at)
    .bind(&created_at)
    .bind(&updated_at)
    .execute(&state.pool)
    .await
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
        created_at,
        updated_at,
    }))
}

async fn complete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    update_task_completion(&state.pool, &task_id, true).await
}

async fn reopen_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    update_task_completion(&state.pool, &task_id, false).await
}

async fn delete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<()>>, (StatusCode, Json<ErrorResponse>)> {
    let result = sqlx::query(
        r#"
        DELETE FROM tasks
        WHERE id = ?
        "#,
    )
    .bind(&task_id)
    .execute(&state.pool)
    .await
    .map_err(internal_error)?;

    if result.rows_affected() == 0 {
        return Err((
            StatusCode::NOT_FOUND,
            Json(ErrorResponse {
                message: "Task not found".to_string(),
            }),
        ));
    }

    Ok(ok(()))
}

async fn update_task_completion(
    pool: &SqlitePool,
    task_id: &str,
    completed: bool,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    let updated_at = now_iso();
    let completed_at = if completed { Some(now_iso()) } else { None };

    let result = sqlx::query(
        r#"
        UPDATE tasks
        SET completed = ?, completed_at = ?, updated_at = ?
        WHERE id = ?
        "#,
    )
    .bind(if completed { 1 } else { 0 })
    .bind(&completed_at)
    .bind(&updated_at)
    .bind(task_id)
    .execute(pool)
    .await
    .map_err(internal_error)?;

    if result.rows_affected() == 0 {
        return Err((
            StatusCode::NOT_FOUND,
            Json(ErrorResponse {
                message: "Task not found".to_string(),
            }),
        ));
    }

    let row = sqlx::query_as::<
        _,
        (
            String,
            String,
            String,
            i64,
            i64,
            Option<String>,
            Option<String>,
            Option<i64>,
            Option<String>,
            String,
            String,
        ),
    >(
        r#"
        SELECT id, list_id, title, completed, order_index, due_date, recurrence, streak_count, completed_at, created_at, updated_at
        FROM tasks
        WHERE id = ?
        LIMIT 1
        "#,
    )
    .bind(task_id)
    .fetch_one(pool)
    .await
    .map_err(internal_error)?;

    Ok(ok(TaskDto {
        id: row.0,
        list_id: row.1,
        title: row.2,
        completed: row.3 != 0,
        order: row.4,
        due_date: row.5,
        recurrence: row.6,
        streak_count: row.7,
        completed_at: row.8,
        created_at: row.9,
        updated_at: row.10,
    }))
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

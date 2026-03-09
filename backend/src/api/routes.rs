use crate::infrastructure::schema::{lists, task_completions, tasks};
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

// Database model for lists
#[derive(Queryable, Selectable, Serialize)]
#[diesel(table_name = lists)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct List {
    pub id: Uuid,
    pub parent_id: Option<Uuid>,
    pub name: String,
    pub list_type: i16,
}

// Insertable list
#[derive(Insertable)]
#[diesel(table_name = lists)]
pub struct NewList<'a> {
    pub id: &'a Uuid,
    pub parent_id: Option<&'a Uuid>,
    pub name: &'a str,
    pub list_type: i16,
}

// Database model for tasks
#[derive(Queryable, Selectable, Serialize)]
#[diesel(table_name = tasks)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct Task {
    pub id: Uuid,
    pub list_id: Uuid,
    pub text: String,
    pub done: bool,
    pub repeat_on: Option<Vec<Option<i16>>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

// Insertable task
#[derive(Insertable)]
#[diesel(table_name = tasks)]
pub struct NewTask<'a> {
    pub id: &'a Uuid,
    pub list_id: &'a Uuid,
    pub text: &'a str,
    pub done: bool,
    pub repeat_on: Option<&'a Vec<Option<i16>>>,
    pub created_at: &'a DateTime<Utc>,
    pub updated_at: &'a DateTime<Utc>,
    pub deleted_at: Option<&'a DateTime<Utc>>,
}

// Database model for task completions
#[derive(Queryable, Selectable, Serialize)]
#[diesel(table_name = task_completions)]
#[diesel(check_for_backend(diesel::pg::Pg))]
pub struct TaskCompletion {
    pub task_id: Uuid,
    pub date: DateTime<Utc>,
}

// Insertable task completion
#[derive(Insertable)]
#[diesel(table_name = task_completions)]
pub struct NewTaskCompletion<'a> {
    pub task_id: &'a Uuid,
    pub date: &'a DateTime<Utc>,
}

// API DTOs for frontend compatibility
#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ListDto {
    id: String,
    parent_id: Option<String>,
    name: String,
    r#type: String, // "todo", "daily", or "collection"
    config: Value,  // Empty object for now
    created_at: String,
    updated_at: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CreateListRequest {
    parent_id: Option<String>,
    name: String,
    r#type: String, // "todo", "daily", or "collection"
    config: Value,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
pub struct TaskDto {
    id: String,
    list_id: String,
    title: String,                // Maps to 'text' in database
    completed: bool,              // Maps to 'done' in database
    order: i32,                   // Not in schema, default to 0
    due_date: Option<String>,     // Not in schema
    recurrence: Option<String>,   // Maps to 'repeat_on' in database
    streak_count: Option<i32>,    // Not in schema
    completed_at: Option<String>, // From task_completions table
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
        .order_by(lists::name.asc())
        .load(&mut conn)
        .map_err(internal_error)?;

    let lists = results
        .into_iter()
        .map(|list| {
            // Convert list_type integer to string
            let type_str = match list.list_type {
                0 => "todo".to_string(),
                1 => "daily".to_string(),
                2 => "collection".to_string(),
                _ => "todo".to_string(), // default
            };

            ListDto {
                id: list.id.to_string(),
                parent_id: list.parent_id.map(|id| id.to_string()),
                name: list.name,
                r#type: type_str,
                config: Value::Object(Default::default()), // Empty config for now
                created_at: "".to_string(),                // Not in schema
                updated_at: "".to_string(),                // Not in schema
            }
        })
        .collect();

    Ok(ok(lists))
}

async fn create_list(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateListRequest>,
) -> Result<Json<ApiResponse<ListDto>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let id = Uuid::new_v4();

    // Convert type string to integer
    let list_type = match request.r#type.as_str() {
        "todo" => 0,
        "daily" => 1,
        "collection" => 2,
        _ => 0, // default to todo
    };

    // Parse parent_id if provided
    let parent_id_uuid = request
        .parent_id
        .as_ref()
        .and_then(|pid| Uuid::parse_str(pid).ok());

    let new_list = NewList {
        id: &id,
        parent_id: parent_id_uuid.as_ref(),
        name: &request.name,
        list_type,
    };

    diesel::insert_into(lists::table)
        .values(&new_list)
        .execute(&mut conn)
        .map_err(internal_error)?;

    Ok(ok(ListDto {
        id: id.to_string(),
        parent_id: request.parent_id,
        name: request.name,
        r#type: request.r#type,
        config: request.config,
        created_at: "".to_string(),
        updated_at: "".to_string(),
    }))
}

async fn get_tasks_for_list(
    State(state): State<Arc<AppState>>,
    Path(list_id): Path<String>,
) -> Result<Json<ApiResponse<Vec<TaskDto>>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    // Parse list_id as Uuid
    let list_uuid = Uuid::parse_str(&list_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid list ID format".to_string(),
            }),
        )
    })?;

    let results: Vec<Task> = tasks::table
        .filter(tasks::list_id.eq(list_uuid))
        .filter(tasks::deleted_at.is_null()) // Only non-deleted tasks
        .order_by(tasks::created_at.asc())
        .load(&mut conn)
        .map_err(internal_error)?;

    // Get task completions for these tasks
    let task_ids: Vec<Uuid> = results.iter().map(|task| task.id).collect();

    let completions: Vec<TaskCompletion> = if !task_ids.is_empty() {
        task_completions::table
            .filter(task_completions::task_id.eq_any(&task_ids))
            .order_by(task_completions::date.desc())
            .load(&mut conn)
            .map_err(internal_error)?
    } else {
        Vec::new()
    };

    // Group completions by task_id
    use std::collections::HashMap;
    let mut completions_by_task: HashMap<Uuid, Vec<DateTime<Utc>>> = HashMap::new();
    for completion in completions {
        completions_by_task
            .entry(completion.task_id)
            .or_insert_with(Vec::new)
            .push(completion.date);
    }

    let tasks = results
        .into_iter()
        .map(|task| {
            // Get most recent completion for this task
            let completed_at = completions_by_task
                .get(&task.id)
                .and_then(|dates| dates.first())
                .map(|date| date.to_rfc3339());

            // Convert repeat_on array to recurrence string
            let recurrence = task.repeat_on.as_ref().map(|repeat_on| {
                // Convert array of weekday numbers to string representation
                // This is a simplified conversion
                format!("{:?}", repeat_on)
            });

            TaskDto {
                id: task.id.to_string(),
                list_id: task.list_id.to_string(),
                title: task.text,     // Map text to title
                completed: task.done, // Map done to completed
                order: 0,             // Default order since not in schema
                due_date: None,       // Not in schema
                recurrence,
                streak_count: None, // Not in schema
                completed_at,
                created_at: task.created_at.to_rfc3339(),
                updated_at: task.updated_at.to_rfc3339(),
            }
        })
        .collect();

    Ok(ok(tasks))
}

async fn create_task(
    State(state): State<Arc<AppState>>,
    Json(request): Json<CreateTaskRequest>,
) -> Result<Json<ApiResponse<TaskDto>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    let id = Uuid::new_v4();
    let created_at = Utc::now();
    let updated_at = created_at;

    // Parse list_id as Uuid
    let list_id = Uuid::parse_str(&request.list_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid list ID format".to_string(),
            }),
        )
    })?;

    // Convert recurrence string to repeat_on array (simplified)
    let repeat_on = request.recurrence.as_ref().map(|_| {
        // This is a simplified conversion - in a real app you'd parse the recurrence string
        // For now, return an empty array
        vec![]
    });

    let new_task = NewTask {
        id: &id,
        list_id: &list_id,
        text: &request.title, // Map title to text
        done: request.completed,
        repeat_on: repeat_on.as_ref().map(|ro| ro),
        created_at: &created_at,
        updated_at: &updated_at,
        deleted_at: None,
    };

    diesel::insert_into(tasks::table)
        .values(&new_task)
        .execute(&mut conn)
        .map_err(internal_error)?;

    // If task is completed and completed_at is provided, create a task completion record
    if request.completed {
        if let Some(completed_at_str) = &request.completed_at {
            if let Ok(completed_at) = DateTime::parse_from_rfc3339(completed_at_str) {
                let completed_at_utc = completed_at.with_timezone(&Utc);
                let new_completion = NewTaskCompletion {
                    task_id: &id,
                    date: &completed_at_utc,
                };

                diesel::insert_into(task_completions::table)
                    .values(&new_completion)
                    .execute(&mut conn)
                    .map_err(internal_error)?;
            }
        } else {
            // If no completed_at provided, use current time
            let new_completion = NewTaskCompletion {
                task_id: &id,
                date: &created_at,
            };

            diesel::insert_into(task_completions::table)
                .values(&new_completion)
                .execute(&mut conn)
                .map_err(internal_error)?;
        }
    }

    Ok(ok(TaskDto {
        id: id.to_string(),
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

    // Parse task_id as Uuid
    let task_uuid = Uuid::parse_str(&task_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid task ID format".to_string(),
            }),
        )
    })?;

    let updated_at = Utc::now();

    // Update the task's done status
    let num_updated = diesel::update(tasks::table.filter(tasks::id.eq(task_uuid)))
        .set((tasks::done.eq(completed), tasks::updated_at.eq(updated_at)))
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

    // Handle task completions table
    if completed {
        // Add a completion record
        let new_completion = NewTaskCompletion {
            task_id: &task_uuid,
            date: &updated_at,
        };

        diesel::insert_into(task_completions::table)
            .values(&new_completion)
            .execute(&mut conn)
            .map_err(internal_error)?;
    } else {
        // Remove completion records (simplified - in real app might want different logic)
        // We can't compare date directly with DateTime.date(), so we'll remove all completions for this task
        diesel::delete(task_completions::table.filter(task_completions::task_id.eq(task_uuid)))
            .execute(&mut conn)
            .map_err(internal_error)?;
    }

    // Get the updated task
    let task: Task = tasks::table
        .filter(tasks::id.eq(task_uuid))
        .first(&mut conn)
        .map_err(internal_error)?;

    // Get most recent completion for this task
    let completion: Option<TaskCompletion> = task_completions::table
        .filter(task_completions::task_id.eq(task_uuid))
        .order_by(task_completions::date.desc())
        .first(&mut conn)
        .optional()
        .map_err(internal_error)?;

    let completed_at = completion.map(|c| c.date.to_rfc3339());

    // Convert repeat_on array to recurrence string
    let recurrence = task
        .repeat_on
        .as_ref()
        .map(|repeat_on| format!("{:?}", repeat_on));

    Ok(ok(TaskDto {
        id: task.id.to_string(),
        list_id: task.list_id.to_string(),
        title: task.text,
        completed: task.done,
        order: 0,
        due_date: None,
        recurrence,
        streak_count: None,
        completed_at,
        created_at: task.created_at.to_rfc3339(),
        updated_at: task.updated_at.to_rfc3339(),
    }))
}

async fn delete_task(
    State(state): State<Arc<AppState>>,
    Path(task_id): Path<String>,
) -> Result<Json<ApiResponse<()>>, (StatusCode, Json<ErrorResponse>)> {
    let mut conn = state.pool.get().map_err(internal_error)?;

    // Parse task_id as Uuid
    let task_uuid = Uuid::parse_str(&task_id).map_err(|_| {
        (
            StatusCode::BAD_REQUEST,
            Json(ErrorResponse {
                message: "Invalid task ID format".to_string(),
            }),
        )
    })?;

    // Soft delete by setting deleted_at timestamp
    let updated_at = Utc::now();
    let num_updated = diesel::update(tasks::table.filter(tasks::id.eq(task_uuid)))
        .set((
            tasks::deleted_at.eq(Some(updated_at)),
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

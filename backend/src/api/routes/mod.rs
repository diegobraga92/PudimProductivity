pub mod lists;
pub mod tasks;

use crate::infrastructure::repositories::{
    list_repository::ListRepository, task_repository::TaskRepository,
};
use axum::Router;
use std::sync::Arc;
use tower_http::cors::{Any, CorsLayer};

pub type DbPool = crate::infrastructure::repositories::list_repository::DbPool;

#[derive(Clone)]
pub struct AppState {
    pub pool: DbPool,
    pub list_repository: ListRepository,
    pub task_repository: TaskRepository,
}

impl AppState {
    pub fn new(pool: DbPool) -> Self {
        Self {
            pool: pool.clone(),
            list_repository: ListRepository::new(pool.clone()),
            task_repository: TaskRepository::new(pool),
        }
    }
}

pub fn create_router(state: AppState) -> Router {
    let state = Arc::new(state);

    Router::new()
        .route("/lists", get(lists::get_lists).post(lists::create_list))
        .route("/lists/:list_id/tasks", get(lists::get_tasks_for_list))
        .route("/tasks", post(tasks::create_task))
        .route("/tasks/:task_id", delete(tasks::delete_task))
        .route("/tasks/:task_id/complete", patch(tasks::complete_task))
        .route("/tasks/:task_id/reopen", patch(tasks::reopen_task))
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_methods(Any)
                .allow_headers(Any),
        )
        .with_state(state)
}

use axum::routing::{delete, get, patch, post};

// TODO REVIEW

mod api;
mod domain;
mod infrastructure;

use api::routes::{create_router, AppState};
use infrastructure::db::init_db;
use tokio::net::TcpListener;

#[tokio::main]
async fn main() {
    // Initialize DB
    let pool = init_db();

    let state = AppState { pool };
    let app = create_router(state);

    // Bind listener
    let listener = TcpListener::bind("0.0.0.0:3000")
        .await
        .expect("Failed to bind address");

    println!("Server running on http://0.0.0.0:3000");

    axum::serve(listener, app).await.expect("Server failed");
}
// TODO

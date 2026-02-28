use sqlx::{
    sqlite::{SqliteConnectOptions, SqlitePoolOptions},
    SqlitePool,
};
use std::str::FromStr;

pub async fn init_db(database_url: &str) -> SqlitePool {
    let options = SqliteConnectOptions::from_str(database_url)
        .expect("Invalid SQLite database URL")
        .create_if_missing(true);

    let pool = SqlitePoolOptions::new()
        .max_connections(5)
        .connect_with(options)
        .await
        .expect("Failed to connect to DB");

    // Create tables if not exist
    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS events (
            id TEXT PRIMARY KEY,
            entity_id TEXT NOT NULL,
            entity_type TEXT NOT NULL,
            event_type TEXT NOT NULL,
            payload TEXT NOT NULL,
            timestamp INTEGER NOT NULL,
            device_id TEXT NOT NULL
        );
        "#,
    )
    .execute(&pool)
    .await
    .unwrap();

    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            email TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL
        );
        "#,
    )
    .execute(&pool)
    .await
    .unwrap();

    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS lists (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            name TEXT NOT NULL,
            type TEXT NOT NULL,
            config TEXT NOT NULL,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL
        );
        "#,
    )
    .execute(&pool)
    .await
    .unwrap();

    sqlx::query(
        r#"
        CREATE TABLE IF NOT EXISTS tasks (
            id TEXT PRIMARY KEY,
            list_id TEXT NOT NULL,
            title TEXT NOT NULL,
            completed INTEGER NOT NULL,
            order_index INTEGER NOT NULL,
            due_date TEXT,
            recurrence TEXT,
            streak_count INTEGER,
            completed_at TEXT,
            created_at TEXT NOT NULL,
            updated_at TEXT NOT NULL
        );
        "#,
    )
    .execute(&pool)
    .await
    .unwrap();

    // Seed a default user for local development login.
    sqlx::query(
        r#"
        INSERT OR IGNORE INTO users (id, email, password)
        VALUES ('00000000-0000-0000-0000-000000000001', 'admin@local', 'admin')
        "#,
    )
    .execute(&pool)
    .await
    .unwrap();

    // Ensure fixed Daily and To Do lists exist for frontend expectations.
    sqlx::query(
        r#"
        INSERT OR IGNORE INTO lists
            (id, user_id, name, type, config, created_at, updated_at)
        VALUES
            ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Daily', 'daily', '{"showCompleted":false,"daysOfWeek":[0,1,2,3,4,5,6],"analyticsEnabled":true}', datetime('now'), datetime('now')),
            ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'To Do', 'todo', '{"showCompleted":false,"autoArchive":false}', datetime('now'), datetime('now'))
        "#,
    )
    .execute(&pool)
    .await
    .unwrap();

    pool
}
// TODO

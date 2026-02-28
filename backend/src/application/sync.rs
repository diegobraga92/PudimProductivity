use crate::domain::event::Event;
use sqlx::SqlitePool;

pub async fn insert_event(pool: &SqlitePool, event: &Event) {
    sqlx::query(
        r#"
        INSERT INTO events 
        (id, entity_id, entity_type, event_type, payload, timestamp, device_id)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        "#,
    )
    .bind(event.id.to_string())
    .bind(event.entity_id.to_string())
    .bind(&event.entity_type)
    .bind(&event.event_type)
    .bind(event.payload.to_string())
    .bind(event.timestamp)
    .bind(event.device_id.to_string())
    .execute(pool)
    .await
    .unwrap();
}

pub async fn get_events_since(pool: &SqlitePool, since: i64) -> Vec<Event> {
    let rows = sqlx::query_as::<_, (String, String, String, String, String, i64, String)>(
        r#"
        SELECT id, entity_id, entity_type, event_type, payload, timestamp, device_id
        FROM events
        WHERE timestamp > ?
        ORDER BY timestamp ASC
        "#,
    )
    .bind(since)
    .fetch_all(pool)
    .await
    .unwrap();

    rows.into_iter()
        .map(|row| Event {
            id: row.0.parse().unwrap(),
            entity_id: row.1.parse().unwrap(),
            entity_type: row.2,
            event_type: row.3,
            payload: serde_json::from_str(&row.4).unwrap(),
            timestamp: row.5,
            device_id: row.6.parse().unwrap(),
        })
        .collect()
}
// TODO

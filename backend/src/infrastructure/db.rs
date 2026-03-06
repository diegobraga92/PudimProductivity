use diesel::pg::PgConnection;
use diesel::r2d2::{ConnectionManager, Pool, PoolError};
use std::env;
use std::time::Duration;

pub type DbPool = Pool<ConnectionManager<PgConnection>>;

pub fn init_db() -> Result<DbPool, PoolError> {
    let database_url = env::var("DATABASE_URL").expect("DATABASE_URL must be set");

    let max_size: u32 = env::var("DB_POOL_SIZE")
        .unwrap_or_else(|_| "15".into())
        .parse()
        .expect("DB_POOL_SIZE must be a number");

    let manager = ConnectionManager::<PgConnection>::new(database_url);

    Pool::builder()
        .max_size(max_size)
        .connection_timeout(Duration::from_secs(5))
        .build(manager)
}

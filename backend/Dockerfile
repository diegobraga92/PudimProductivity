FROM rust:1.86-bookworm AS builder

WORKDIR /app/backend

COPY backend/Cargo.toml ./Cargo.toml
COPY backend/src ./src

RUN cargo install diesel_cli --no-default-features --features postgres

RUN cargo build --release

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        postgresql-client \
        libpq-dev \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/backend/target/release/backend /usr/local/bin/backend
COPY --from=builder /usr/local/cargo/bin/diesel /usr/local/bin/diesel

EXPOSE 3000

CMD ["backend"]
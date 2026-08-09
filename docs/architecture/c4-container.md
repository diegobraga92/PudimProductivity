# C4 — Container Diagram

> Level 2 of the C4 model. Shows the top-level containers inside the backend
> system and the data flows between them.

## Diagram

```mermaid
flowchart TB
    subgraph CLIENT [Clients]
        WEB[Web SPA<br/>React + React Query]
        MOB[Android App<br/>Kotlin + Compose]
    end

    subgraph BE [Backend — Go modular monolith (single process)]
        direction TB
        ROUTER[HTTP Router<br/>chi + middleware]
        TASK[Task Module<br/>service + postgres repo]
        TLIST[TaskList Module]
        PLAN[Planner Module]
        POMO[Pomodoro Module<br/>in-memory sessions]
        FF[FeatureFlag Module]
        SYNC[Sync Hub<br/>WebSocket + replay buffer]
        NOTIF[Notification Worker<br/>consumes RabbitMQ]
        AUDIT[Audit Module]
        METRICS[Prometheus Metrics :9090]
        OTEL[OTEL Tracing<br/>OTLP/HTTP exporter]

        ROUTER --> TASK & TLIST & PLAN & POMO & FF & SYNC
        TASK --> AUDIT
        TASK --> EBUS
    end

    subgraph EBUS [Event Bus]
        direction LR
        INMEM[InMemory Bus<br/>sync fan-out]
        AMQPBUS[RabbitMQ Bus<br/>durable fan-out]
        COMP[CompositeBus]
    end

    TASK --> COMP
    COMP --> INMEM
    COMP --> AMQPBUS
    INMEM --> SYNC
    AMQPBUS --> NOTIF

    WEB -->|"HTTPS /api/v1"| ROUTER
    MOB -->|"HTTPS /api/v1"| ROUTER
    WEB -->|"WSS /api/v1/ws"| SYNC
    MOB -->|"WSS /api/v1/ws"| SYNC

    TASK -->|"SQL"| PG[(PostgreSQL)]
    TLIST -->|"SQL"| PG
    PLAN -->|"SQL"| PG
    FF -->|"SQL"| PG
    AUDIT -->|"SQL"| PG
    TASK -->|"cache"| REDIS[(Redis)]

    AMQPBUS -->|"AMQP"| AMQP[(RabbitMQ)]
    NOTIF -->|"SMTP"| MAILPIT[Mailpit]
    NOTIF -->|"FCM"| FCM[Firebase]

    ROUTER --> METRICS
    ROUTER --> OTEL
```

## Containers

| Container | Responsibility |
|-----------|----------------|
| **HTTP Router** | chi-based routing, `RequestID`, auth/RBAC middleware (dev headers), timeout, metrics + tracing middleware. |
| **Task Module** | Task + habit + completion CRUD, streak-ready read models, audit logging. |
| **TaskList Module** | Named task collections. |
| **Planner Module** | Weekly scheduling fields on tasks (`start_time`, `end_time`, `color`, `scheduled_date`). |
| **Pomodoro Module** | In-memory focus timer sessions (state is not persisted by design). |
| **FeatureFlag Module** | Admin-toggleable flags served to clients. |
| **Sync Hub** | WebSocket endpoint, per-process monotonic `seq`, replay ring buffer, stale→refetch protocol (ADR 004). |
| **Notification Worker** | Consumes `task.events` from RabbitMQ, dedupes via `notifications` table, sends email/push (ADR 005). |
| **Audit Module** | Append-only audit log for task CRUD, pomodoro sessions, flag toggles. |
| **Event Bus** | `CompositeBus` fans to in-memory (real-time) + RabbitMQ (durable async), never blocking the request path. |
| **Metrics / Tracing** | Prometheus histograms + counters on internal `:9090`; W3C trace context through HTTP → event bus → RabbitMQ headers → logs. |

## Deployment Shape

- **Local / LAN (default):** everything in `docker-compose.yml` on one host —
  postgres, rabbitmq, mailpit, backend, web (nginx), optional jaeger/prometheus/
  grafana profiles. See `README.md`.
- **CI:** backend + fresh Postgres service container only; RabbitMQ/Redis are
  optional (backend starts without them).
- **Production target (future):** single-host compose is the MVP deployment;
  multi-node orchestration with GitOps is designed in ADR 006.

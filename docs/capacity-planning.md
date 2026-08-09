# Capacity Planning & Cost Report

> Estimates for hosting PudimProductivity on AWS (single region, `us-east-1`).
> Prices are approximate list prices (2026) and region-dependent — treat as
> **planning numbers**, not quotes. Units: monthly unless noted.

## 1. Current Scale & Measured Capacity

- **Product shape:** single-user productivity app (tasks, habits, planner,
  pomodoro, notifications) on a modular-monolith backend.
- **Workload model:** a handful of reads/writes per active minute; bursts when
  the user opens the app (list + habit queries + completions).
- **Measured headroom (Phase 6 load tests, single dev machine):**
  - 50 VUs sustained ≈ **77 req/s** against the task API at **p95 ≈ 25 ms**
    (tasks) / 36 ms (habits), 0% errors.
  - DB: 50k rows; habit-list query uses the partial index; pgbench baseline
    **982 TPS** at 10 clients on a laptop Postgres.
- **Runtime footprint measured locally:**
  - Backend container: ~20–30 MB resident memory, < 0.5 vCPU at idle-to-light.
  - Postgres 16 (single database): < 100 MB for MVP data.
  - Web: static nginx (single-page app) — negligible.

## 2. Growth Scenarios

| Scenario | Users | Sustained req/s | Peak req/s | DB rows (tasks+completions) | Notes |
|----------|-------|-----------------|------------|------------------------------|-------|
| **S0 — MVP** | 1 | ~1–5 | ~20 | ~10⁴ | Current |
| **S1 — 10×** | 100 DAU | ~50 | ~200 | ~10⁶ | Family/team or public beta |
| **S2 — 100×** | 1 000 DAU | ~500 | ~2 000 | ~10⁷ | Product-market fit |

Read/write mix assumed 80/20. The task API p95 SLO (200 ms) was met at
**≈ 77 req/s** on a single host with zero tuning, so **S1 fits one mid-size
instance today**; S2 needs horizontal scale or stronger caching.

## 3. Capacity Triggers (when to scale)

| Signal | Threshold | Action |
|--------|-----------|--------|
| API p95 latency | > 150 ms for 5 min (SLO is 200 ms) | Add CPU / verify index usage (`pg_stat_statements` + `pg_query_duration_seconds`) |
| CPU saturation | > 70% for 15 min | Scale up vCPU; if writes dominate, move to multi-node with RabbitMQ fan-out |
| DB connections | > 80% of `DATABASE_MAX_CONNS` (20) | Tune pool, add PgBouncer, or scale RDS |
| Redis cache hit rate | < 60% | Revisit TTLs/invalidation (ADR 003) |
| Disk | > 70% of allocated volume | RDS autoscaled storage or manual growth |

## 4. Monthly Cost Estimate (S0 → S1 → S2)

### S0 — MVP (single user, dev-scale)

| Component | Config | Est. $/mo |
|-----------|--------|-----------|
| Backend (ECS Fargate) | 0.25 vCPU / 0.5 GB, 1 task | 6 |
| Web (Fargate + nginx) | 0.25 vCPU / 0.5 GB, 1 task | 6 |
| ALB | 1 small ALB | 18 |
| RDS PostgreSQL | `db.t4g.small` single-AZ, 20 GB gp3 | 15 |
| S3 (static assets) | 5 GB + transfer | 1 |
| CloudFront | 5 GB egress | 1 |
| **Total** | | **≈ $47/mo** |

> Self-hosted LAN (the current default per README) costs **$0/mo** in cloud —
> this estimate is the "put it on AWS properly" path.

### S1 — 10× (100 DAU)

| Component | Config | Est. $/mo |
|-----------|--------|-----------|
| Backend (Fargate) | 1 vCPU / 2 GB ×2 (HA) | 45 |
| Web (Fargate) | 0.25 vCPU / 0.5 GB ×2 | 12 |
| ALB + LCUs | | 25 |
| RDS PostgreSQL | `db.t4g.medium` single-AZ, 100 GB gp3 | 35 |
| ElastiCache Redis | `cache.t4g.small` (optional cache) | 12 |
| Amazon MQ (RabbitMQ) | `mq.t3.micro` | 40 |
| S3 + CloudFront + WAF | | 10 |
| **Total** | | **≈ $180/mo** |

### S2 — 100× (1 000 DAU)

| Component | Config | Est. $/mo |
|-----------|--------|-----------|
| Backend (Fargate) | 2 vCPU / 4 GB ×4, autoscaling | 180 |
| Web (Fargate) | 0.5 vCPU / 1 GB ×2 | 20 |
| ALB + LCUs | | 40 |
| RDS PostgreSQL | `db.m6g.large` Multi-AZ, 500 GB gp3 | 240 |
| ElastiCache Redis | `cache.m6g.large` (cache + sessions) | 100 |
| Amazon MQ (RabbitMQ) | `mq.m5.large` multi-AZ | 250 |
| S3 + CloudFront + WAF | | 25 |
| **Total** | | **≈ $855/mo** |

## 5. Cost Optimization Opportunities

1. **Reserved / Savings Plan RDS + Fargate:** 1-year committed-use cuts compute
   ~30–40%. S0–S1 can skip entirely; S2 should commit for the steady-state base.
2. **Spot Fargate for non-critical tasks:** CI runners, image workers, and the
   notification worker tolerate interruption.
3. **S3 lifecycle policy:** static app assets are immutable and versioned — add
   a 90-day transition to S3-IA for old versions.
4. **Right-size early:** the MVP fits a `db.t4g.small`; do not pre-provision
   for S2. Every tier in this report is *triggered* by the capacity signals in
   §3, not bought upfront.
5. **RabbitMQ → managed or in-process:** at S0 the `CompositeBus` runs without
   RabbitMQ entirely; paying for Amazon MQ only makes sense when async volume
   requires it (S1+).
6. **CloudFront only if global:** for a LAN/local deployment, skip CDN and serve
   nginx directly — saves ~$1–3/mo and one DNS record.

## 6. Cost per 1 000 users (efficiency)

- **S1:** ≈ $1.80 per 1 000 DAU per month
- **S2:** ≈ $0.86 per 1 000 DAU per month

Fixed infrastructure (ALB, RDS base, MQ) dominates at small scale; cost-per-user
roughly halves from S1 to S2 as the fixed base amortises. This is the typical
shape of a single-region modular monolith — **the cheapest possible architecture
at this scale**, which is why ADR 002/006 deliberately avoid microservices and
a cluster for the MVP.

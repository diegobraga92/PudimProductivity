# Service Level Objectives (SLOs)

> This document defines Service Level Indicators (SLIs), SLO targets, and error budgets for the PudimProductivity services. Updated as new services and features are added.

---

## Table of Contents

1. [Health Endpoint SLO](#1-health-endpoint-slo)
2. [Task API SLO (Phase 1)](#2-task-api-slo-phase-1)
3. [Alerting Rules](#3-alerting-rules)
4. [Measurement & Burn Rate](#4-measurement--burn-rate)

---

## 1. Health Endpoint SLO

**SLI:** `GET /api/v1/health` — success rate of HTTP 200 responses.

**SLO Target:** 99.5% availability over a 30-day rolling window.

**Error Budget Calculation:**

| Window | Total requests | Allowed failures (0.5%) |
|--------|----------------|------------------------|
| 30 days | — | ~3 hours 36 minutes of downtime |
| 7 days  | — | ~50 minutes of downtime |
| 1 day   | — | ~7 minutes of downtime |

**Measurement:**

- Metric: `http_requests_total{handler="/api/v1/health", status=~"2.."} / http_requests_total{handler="/api/v1/health"}`
- Aggregation: 1-minute rate, averaged over the rolling window
- Source: Prometheus (once metrics endpoint is deployed)

**Dependencies:** PostgreSQL, Go runtime

---

## 2. Task API SLO (Phase 1)

**SLI:** `GET/POST/PUT/DELETE /api/v1/tasks*` — success rate of HTTP 2xx responses and p95 latency.

**SLO Targets:**

| Indicator | Target | Window |
|-----------|--------|--------|
| Success rate | 99.0% | 30-day rolling |
| Latency (p95) | < 200ms | 30-day rolling |

**Error Budget (Success Rate):**

| Window | Allowed failures (1.0%) |
|--------|------------------------|
| 30 days | ~7 hours 12 minutes |
| 7 days  | ~1 hour 40 minutes |
| 1 day   | ~14 minutes |

**Measurement:**

- Success rate: `sum(rate(http_requests_total{handler=~"/api/v1/tasks.*", status=~"2.."}[1m])) / sum(rate(http_requests_total{handler=~"/api/v1/tasks.*"}[1m]))`
- Latency: `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{handler=~"/api/v1/tasks.*"}[5m])) by (le))`
- Source: Prometheus via `http_request_duration_seconds` histogram (see Phase 1 item 3)

**Dependencies:** PostgreSQL, Redis (when caching is enabled)

---

## 3. Alerting Rules

### Burn Rate Alerts

Burn rate measures how fast the error budget is being consumed relative to the SLO.

| Severity | Burn Rate | Alert Condition | Response Time |
|----------|-----------|-----------------|---------------|
| Critical | 2% budget/hour | `burn_rate_2p_1h > 0` | 15 minutes |
| Warning  | 5% budget/6h  | `burn_rate_5p_6h > 0` | 30 minutes |
| Info     | 10% budget/24h | `burn_rate_10p_24h > 0` | 2 hours |

### Prometheus Alerting Rules (to be added to `infra/prometheus/alerts.yml`)

```yaml
groups:
  - name: slo-alerts
    interval: 30s
    rules:
      # Health endpoint burn rate
      - alert: HealthEndpointHighErrorRate
        expr: |
          (
            sum(rate(http_requests_total{handler="/api/v1/health", status!~"2.."}[1m]))
            / sum(rate(http_requests_total{handler="/api/v1/health"}[1m]))
          ) > 0.005
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Health endpoint error rate exceeds SLO"

      # Task API burn rate
      - alert: TaskApiHighErrorRate
        expr: |
          (
            sum(rate(http_requests_total{handler=~"/api/v1/tasks/.*", status!~"2.."}[1m]))
            / sum(rate(http_requests_total{handler=~"/api/v1/tasks/.*"}[1m]))
          ) > 0.01
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Task API error rate exceeds SLO"

      - alert: TaskApiHighLatency
        expr: |
          histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{handler=~"/api/v1/tasks/.*"}[5m])) by (le))
          > 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Task API p95 latency exceeds 200ms"
```

---

## 4. Measurement & Burn Rate

### Implementation Details

- **Metrics are scraped from** `backend:9090/metrics` (internal port, not exposed publicly).
- **Alerting is configured in** Prometheus `infra/prometheus/alerts.yml`.
- **Dashboards in** Grafana with RED metrics for each service endpoint.

### Burn Rate Calculation

Burn rate is defined as the ratio of actual error rate to SLO-allowed error rate:

```
burn_rate = actual_error_rate / (1 - slo_target)
```

- `burn_rate = 1.0` → budget consumed exactly at expected rate (steady state)
- `burn_rate > 1.0` → budget being consumed faster than expected
- `burn_rate > 10` → budget would be exhausted in 1/10th of the SLO window

### Multi-window, Multi-burn-rate Alerting

We use the Google SRE approach of multi-window, multi-burn-rate alerts:
- Short window (1h, 5% burn rate) for fast detection of outages.
- Long window (6h, 10% burn rate) to catch slow-burn issues (e.g., gradual latency increase).
- Both conditions must fire simultaneously to trigger an alert (reduces false positives).

---

## Appendix: SLO Review Cadence

- **Monthly:** Review error budget consumption. If budget is consistently depleted, adjust SLO targets or allocate engineering time to improve reliability.
- **Quarterly:** Update this document with new services, revised targets, and lessons from incidents.
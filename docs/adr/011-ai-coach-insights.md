# ADR 011: AI Coach — Template-First Weekly Reports with Optional LLM

**Status:** Accepted
**Date:** 2026-08-10
**Author:** Diego Braga

## Context

Phase 9a adds an "AI coach": a weekly productivity report generated from
existing domain events (task completions, pomodoro focus sessions, recipe
creations). Three design questions:

1. **How are reports generated** — a deterministic template, a large language
   model, or a hybrid?
2. **How does the module get its data** — query the DB on demand, or consume
   events?
3. **Where does the user-facing surface live** — server-generated prose,
   structured JSON for the client, or both?

## Decision

### Template-first, LLM optional behind a feature flag

- `internal/insights` renders a **Go `text/template`** into plain-English prose
  (`report_text`) from structured `WeeklyStats` (total completions, per-day
  rate, top-3 habits, focus minutes/sessions, new recipes).
- An **optional LLM summary** is produced only when the `insights.llm_enabled`
  feature flag is on **and** a `Summarizer` is configured. The default is
  `NoopSummarizer` (zero external calls, zero API key, zero cost). Failures are
  logged and fall back to the template text — the LLM can never break the
  report.

Rationale: deterministic reports are testable, free, and predictable; the LLM
adds perceived intelligence but not correctness. Gating it behind a feature
flag (already in the codebase since Phase 1) makes the experiment cheap to
turn on/off per deployment.

### Consume events for focus history; query for everything else

- **Pomodoro** sessions are in-memory (no persistence), so the insights module
  **subscribes to `pomodoro.session.completed`** and persists completed
  sessions into `pomodoro_sessions`. The write runs on its own goroutine so a
  slow DB write never blocks the in-memory bus (per the eventbus contract).
- **Completions** and **recipes** already live in Postgres; the report queries
  them directly for the target week.

This keeps a single writer per table (the event consumer) and avoids a second
write path through the pomodoro module.

### Both prose and structured JSON

The report is stored with `report_text` (for humans) and `report_json` (for the
web/mobile stat cards). Clients can render either without re-deriving data.

## Alternatives Considered

- **Pure LLM generation**: non-deterministic, expensive, un-testable, and
  depends on an external service for a core feature. Rejected for the primary
  path.
- **Event-driven materialization of a full profile**: the scheduler (ADR 009)
  already derives its profile on demand; insights reuses that philosophy —
  aggregations are cheap (< 4 indexed queries per week).
- **Streaming analytics**: overkill for a weekly report.

## Consequences

- Reports are cacheable per `(user_id, week_start)` in `insight_reports`;
  regeneration only happens when no cached copy exists.
- New `pomodoro_sessions` table fills the pomodoro persistence gap and will
  also feed Phase 9c's offline sync and future analytics.
- Privacy: the LLM (when enabled) receives only the aggregated template text —
  never raw task titles or event payloads. This is documented for the
  stakeholder-facing README.

## When to Revisit

- If users want daily or real-time insights, the per-`(user,week)` cache key
  should become per-day and the aggregation window parameterized.
- When a real LLM provider is configured, add the OpenAI-compatible client and
  a `Summarizer` implementation behind the same feature flag (the interface is
  already in place).

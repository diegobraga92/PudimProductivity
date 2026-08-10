# ADR 009: Auto-Scheduler with Derived Profile

**Status:** Accepted
**Date:** 2026-08-09
**Author:** Diego Braga

## Context

Phase 7 adds an auto-scheduler: "suggest a plan for my day". The scheduler
needs two inputs:

1. **A user profile** — when they typically work, how much they complete.
2. **A placement algorithm** — fit pending work into free time.

Two design questions drive the decision:

- Should the profile be **persisted** (updated incrementally from the event
  bus) or **derived on demand** from history?
- Should the scheduler **write** suggested slots back to the task table, or
  return them as **read-only suggestions**?

## Decision

### Profile is derived, not persisted

`internal/scheduler` computes `UserProfile` as a pure function of the last 14
days of completion history on each request:

- **Work window**: min/max completion hour (clamped to 07:00–21:00),
  defaulting to 09:00–18:00 when no history exists.
- **Productivity**: average completions per day.

Rationale: the profile changes slowly, the data already lives in Postgres, and
a derived profile can never go stale or drift out of sync. A persisted profile
would buy incremental O(1) reads at the cost of a write path, invalidation
logic, and consistency bugs.

### Suggestions are read-only

`GET /api/v1/schedule` returns a `Suggestion` — a list of `ScheduleSlot`s
(habit first, then pending todos) fitted into free blocks that exclude
existing planner entries. The client decides whether to accept them; a future
"Apply" action can persist via the existing task update endpoint.

The **placement algorithm**:

1. Load pending one-off todos, all habits, and scheduled (time-blocked) tasks.
2. Build occupied intervals for the date (planned entries that recur that
   weekday or are scheduled for that date).
3. Compute free blocks inside the derived work window.
4. Assign each habit/todo a `defaultDurationMinutes` (30) block from the first
   free block large enough.

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Derived profile + read-only suggestions** | Never stale; no write path; simple | Recomputes profile each call (cheap: 14-day history query) |
| **Persisted profile via event bus** | O(1) profile reads | Staleness, invalidation, extra state |
| **Scheduler writes slots to tasks** | One-tap apply | Mutates user data without consent; needs rollback/undo |

## Consequences

**Positive:**

- The scheduler is stateless and trivially testable with a fake `TaskReader`.
- No schema migration, no event-bus coupling — the ADR-004/005 machinery is
  untouched.
- Habit-first ordering surfaces recurring commitments before one-off todos.

**Negative:**

- The "next free block" greedy algorithm is not optimal (no time-window or
  deadline awareness). That is a v2 concern.
- All tasks use a fixed 30-minute estimate; per-task durations are a v2
  enhancement.

## When to Revisit

- When tasks gain explicit duration/deadline fields (Phase 8 follow-up), move
  to a time-window algorithm.
- If "Apply schedule" becomes a product requirement, add a
  `POST /api/v1/schedule/apply` that persists accepted slots atomically.

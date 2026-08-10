# ADR 008: Rule-Based NLP Parser for Task Input

**Status:** Accepted
**Date:** 2026-08-09
**Author:** Diego Braga

## Context

Phase 7 adds natural-language task entry ("Buy milk tomorrow at 9am"). Options
for understanding free text:

1. **Rule-based parser** (regex + keyword extraction): deterministic,
   dependency-free, testable.
2. **LLM-based parsing**: handles arbitrary phrasing, but adds an API-key
   dependency, cost per call, latency, and non-determinism.
3. **Third-party NLP library**: heavier dependency, licensing and version
   churn, opinionated date/entity handling.

The product is a personal productivity app with a known vocabulary of useful
phrases (dates, times, durations, weekday recurrences). The phrase space is
small and well-bounded.

## Decision

Implement a **rule-based parser** in `internal/nlp` and expose it as
`POST /api/v1/tasks/parse` (Phase 7). The parser:

- Extracts, in order: recurrence ("every mon,wed,fri"), duration
  ("for 30 min", "1 hour"), date ("today", "tomorrow", "next friday",
  "aug 20", "in 3 days", ISO), then time ("at 3pm", "14:30", "9:15 am").
- Removes matched phrases and uses the remainder as the title.
- Is **partial by design**: unsupported patterns simply leave fields nil and
  the remaining text stays as the title. Clients pre-fill what they can and
  keep the rest editable.
- Returns `422` only when *nothing* recognizable remains (empty title and no
  extracted fields).

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Rule-based** | Deterministic, free, fast, fully unit-testable | Limited phrasing coverage |
| **LLM parsing** | Natural coverage | Cost/latency/non-determinism; API key dependency |
| **Third-party NLP lib** | Broad coverage | Dependency churn; overkill for the phrase space |

## Consequences

**Positive:**

- Fully unit-tested (10+ canonical phrases) with an injectable clock for
  deterministic relative dates.
- Zero runtime dependencies; CI-free from external services.
- The `Parse` function is clock-injectable and reusable by the scheduler
  module (Block B) for parsing event text.

**Negative:**

- Users who type unusual phrasing get partial results — the UI shows what was
  parsed and lets them correct it.
- Date semantics are opinionated: "next friday" = next week's Friday, a bare
  "friday" = the upcoming one.

## When to Revisit

- If users consistently phrase tasks in ways the grammar misses, evaluate an
  LLM-based fallback behind the same `POST /tasks/parse` interface — the
  endpoint contract keeps clients unchanged.

# ADR 007: External API Integration Pattern

**Status:** Accepted
**Date:** 2026-08-09
**Author:** Diego Braga

## Context

Phase 5 introduces the first external API integration (Google Books) and
direct-to-object-storage uploads (presigned S3 URLs for recipe images). These
are the templates for every future integration (meal-plan image upload,
calendar sync in Phase 7, barcode/ISBN flows). Requirements:

- **Fail fast** when the vendor is down — the backend should not pile requests
  onto a dead service.
- **Stay within quota** — the Google Books anonymous tier is rate-limited;
  a local token bucket is the cheapest guard.
- **Bound resources** — third-party payloads and latencies must not exhaust
  memory or request time.
- **Graceful degradation** — an integration failure must not break the rest of
  the app (a 502 on `/books/by-isbn` must not affect tasks).
- **Testable without the vendor** — adapters must run against stubs in CI.

## Decision

### 1. One shared HTTP client (`internal/httpclient`)

Every external API adapter uses `internal/httpclient.Client`, which composes:

| Concern | Mechanism |
|---------|-----------|
| Per-request timeout | `http.Client.Timeout` (default 10s) |
| Bounded response body | `io.LimitReader` cap (default 4 MiB) |
| Bounded retries | exponential backoff on network/429/5xx only |
| Rate limiting | token bucket (`Rate`/`Burst`) |
| Circuit breaker | 3 failures → open 30s → half-open probe (fail-fast) |

### 2. Adapters are thin, testable, vendor-scoped

- Adapter packages live beside their domain: `internal/booktrack/googlebooks`.
- The adapter returns a **flat domain DTO** (`googlebooks.BookInfo`), never the
  vendor's raw JSON, so the module is insulated from vendor schema changes.
- Domain modules depend on a **consumer-side interface** (e.g.
  `booktrack.LookupClient`, `mealplan.RecipeReader`), so production can inject
  the real adapter and tests a stub — no interface-package or DI framework.

### 3. Media uploads via presigned URLs, optional at runtime

`internal/media.Generator` issues short-lived S3 presigned PUT URLs; the
client uploads directly, the server never proxies bytes. **No storage
backend → degraded mode** (upload endpoints return 503, everything else works),
matching the graceful-degradation matrix in `docs/graceful-degradation.md`.

### 4. Failure semantics

- Vendor 4xx → mapped to a domain error (`googlebooks.ErrNotFound` → HTTP 404).
- Vendor 5xx/network/quota → wrapped and surfaced as **502** by the handler;
  the circuit breaker prevents retry storms.
- Cross-module dependency failures are isolated: a book lookup failure never
  touches task/recipe/mealplan code.

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Per-adapter hand-rolled HTTP** | No shared abstraction | Retry/breaker/rate code duplicated and untested per adapter |
| **Sidecar / API gateway for integrations** | Scales, centralizes | Overkill for a single-process app; adds a moving part |
| **Vendor SDKs directly in the domain module** | Little glue | Vendor types leak into the domain; hard to stub in tests |
| **Proxy uploads through the server** | No client-side S3 creds | Wastes bandwidth/CPU; presigned URLs are the standard pattern |

## Consequences

**Positive:**

- Google Books is the reference consumer and is fully covered by stub-based
  tests (happy path, not-found, 5xx retry, circuit open, malformed response).
- Every future integration (Phase 7 calendar, meal-plan media) follows the same
  shape: `httpclient` + thin adapter + domain DTO + consumer-side interface.
- Degraded operation is the default until credentials exist.

**Negative:**

- The circuit breaker is in-process (per backend instance) — with multiple
  replicas each trips independently. Acceptable at MVP scale; a shared breaker
  (e.g. via Redis) or API gateway is the migration path.
- Adapters still need real-vendor verification when credentials are available
  (the anonymous Google Books tier was observed returning 429; a key resolves it).

## When to Revisit

- **Multi-replica deployment (ADR 006):** consider a shared rate-limit/breaker
  or an API gateway.
- **Integration count > 3–4:** a dedicated `integrations` sidecar or gateway
  becomes economical.

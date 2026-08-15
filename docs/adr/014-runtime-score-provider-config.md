# ADR 014: Runtime Score-Provider Configuration & Auto-Scoring

**Status:** Accepted
**Date:** 2026-08-15
**Author:** Diego Braga

## Context

The library's rating-provider configuration (which provider serves each media
type, plus API keys) lived only in environment variables
(`SCORE_PROVIDER_MOVIE/SERIES/GAME/BOOK`, `OMDB_API_KEY`, `RAWG_API_KEY`) and
was frozen into the scoring composite at startup. That had two problems:

- Changing a provider or key required editing `.env` and restarting the
  backend — and under `docker compose` the variables weren't even forwarded
  into the backend container.
- The CSV import flow could not fill critic scores automatically, because
  auto-scoring a large list of titles requires a runtime-configurable lookup
  and a batch endpoint.

Requirements:

- Configure providers **at runtime from a UI** (admin page), without restarts.
- Keep the existing per-media-type model (games → RAWG/Metacritic, films and
  series → OMDb/IMDb) — one provider per media type.
- Keep graceful degradation (ADR 007): no credentials/flag off → `503`, provider
  down → `502`, item CRUD never affected.
- API keys are **secrets**: never returned by the API, masked in audit logs,
  excluded from backups.
- Environment variables remain a **one-time bootstrap** for existing deploys.

## Decision

### 1. Database becomes the source of truth

Two tables (`score_providers`, `score_provider_config`) hold the provider
credentials and the media-type → provider mapping. `saved_at` on the singleton
config row is NULL until the user saves via the UI; while NULL the service
overlays the env bootstrap values, so existing `.env` setups keep working on
first start. After the first UI save the DB is authoritative and env is ignored.

New module `backend/internal/scoringsettings/` (repository/service/handler)
mirrors the `featureflag` pattern and exposes admin-only routes:

```
GET /api/v1/admin/score-providers   (masked: api_key_set, never the key)
PUT /api/v1/admin/score-providers   (validate → reload → persist)
```

### 2. Reloadable lookup manager

`scoring.Manager` implements the library's `ScoreLookupProvider` interface
(`Search` + `Configured()`) and holds the current composite behind a mutex.
`Reload(ctx, cfg)` rebuilds the composite and swaps it atomically, so a UI save
takes effect immediately. `Configured()` replaces the old
`lookup.(NoopScoreLookup)` type assertions, which could not survive a wrapper.

### 3. Admin UI + dev role switch

`web/src/pages/ServerSettings.tsx` edits providers/keys and toggles the
`library.score_lookup_enabled` flag. Admin routes require the `admin` role; the
web client stores a dev role in `localStorage` (`devRole`) and sends it as
`X-User-Role`, consistent with the existing dev identity headers. The page
offers the role switch with an explanatory banner.

### 4. Auto-scoring during CSV import

New batch endpoint `POST /api/v1/library/score/batch` (≤100 items, bounded
concurrency, per-item errors inline, same 503/502 semantics). The web CSV
importer gains:

- **⚡ Auto-score** — fills the preview's score + source from each title's top
  candidate so the user can review before importing.
- **Skip review** — runs the same lookups then imports immediately (chunked,
  ≤5000/request), for large files.

Scores already present in the CSV are never overwritten.

### 5. Secrets posture

- `GET` returns only `api_key_set` booleans; `PUT` accepts a new key (blank
  keeps the stored one).
- Audit logs record masked values only (`score_provider.updated`).
- `score_providers` and `score_provider_config` are excluded from backup (the
  backup module is an allowlist, so this is default behavior; the exclusion is
  documented in the code).

## Alternatives Considered

| Approach | Pros | Cons |
|----------|------|------|
| **Keep keys in .env, UI only for mapping/flag** | No secrets in DB | Still requires restarts and env forwarding; does not satisfy the goal |
| **Auto-score inside the import transaction (server-side)** | One round-trip, mobile-friendly | Long synchronous request for large CSVs; no review step; conflicts with the existing "confirm the match" UX |
| **Client loops `score/search` per row** | No new endpoint | N round-trips, no server-side rate limiting/backoff; worse on mobile |
| **Secrets in DB plaintext** | Simplest | Stored secrets; mitigated by write-only API + masked logs + backup exclusion |

## Consequences

**Positive:**

- Provider config and keys are editable at runtime from the UI; no restarts.
- `docker-compose` no longer needs env pass-through for this feature (env only
  bootstraps).
- Auto-scoring makes the "import a CSV of Switch games with critic scores"
  workflow a two-click flow.
- One provider per media type still supports mixed libraries (RAWG for games,
  OMDb for films/series) and new providers are one registry entry away.

**Negative:**

- API keys now live in the database; this changes the "all secrets in env"
  posture (README, security model). Mitigations: write-only API, masked audit,
  no backup inclusion.
- The dev-role switch in the web client is a dev-only affordance — production
  authentication (P0 gap, see threat model) will replace the header trust.

## When to Revisit

- When real user accounts land (P0): replace the dev role switch with actual
  admin authorization.
- If multiple providers per media type (e.g. Metacritic + a second game source
  merged) are ever needed: the composite would grow from `map[type]client` to a
  fan-out.
- If backups should include the media-type mapping (not the keys): add only
  `score_provider_config` to the backup allowlist.

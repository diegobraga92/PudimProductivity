# ADR 013: Home-Screen Widgets for Tasks & Habits

**Status:** Accepted
**Date:** 2026-08-13
**Author:** Diego Braga

## Context

Phase 10 adds Android home-screen widgets so tasks and habits can be seen and
checked off without opening the app. Three design questions:

1. **Which widget framework** — raw `RemoteViews` or Jetpack Glance?
2. **Where does the widget data come from** — the network or the local DB?
3. **How do widgets stay fresh**, and how do taps reach the app?

## Decision

### Jetpack Glance 1.1.1 (Compose for widgets)

Widgets are built with `androidx.glance:glance-appwidget:1.1.1` — the newest
stable release (1.2/1.3 are pre-release only). In 1.1.x the Material 3
components (`CheckBox`, `LinearProgressIndicator`, …) ship inside the
`glance-appwidget` artifact under the `androidx.glance.appwidget` package.

Key constraint satisfied: Glance needs **no KSP/annotation processing**, so it
works with AGP 9.1's built-in Kotlin that blocked Room (ADR 012). A new
toolchain migration was not required.

### Widget data comes from the offline-first local SQLite DB

Widgets run in the app process and read the same `LocalDatabase` (ADR 012)
through a small read-only loader. The render path is instant and works fully
offline — no network call in widget composition. Pure mapping functions
(`buildTasksSnapshot` / `buildHabitsSnapshot`) keep the logic JVM-testable.

### Interactivity: checkboxes + deep links

- **Checkboxes** use `actionRunCallback`: the `ActionCallback` writes the
  change optimistically to the local DB with the same `dirty=1` semantics as
  `TaskRepository` (toggle task status / insert-or-tombstone today's habit
  completion), then enqueues a one-time `SyncWorker` so the server converges.
- **Row taps** use `actionStartActivity`: `MainActivity` reads `EXTRA_SCREEN` /
  `EXTRA_TASK_ID` extras (`singleTop` + `onNewIntent`) and navigates to the
  matching screen — no Navigation library required.

### Refresh strategy

Glance has no reactive update channel, so freshness is pushed:

- `WidgetUpdater.updateAll()` runs after every `TaskRepository.refreshFromLocal`
  (covers local writes, WS-event refreshes and post-sync re-emits) and after
  every successful background sync (`SyncWorker`).
- `updatePeriodMillis` (30 min minimum) in the provider XML + the existing
  15-minute periodic sync act as fallbacks.

## Alternatives Considered

- **Raw `RemoteViews`**: full control and zero new dependencies, but verbose
  and error-prone for list + checkbox widgets with per-row actions.
- **Fetching widget data from the REST API**: slow, fails offline, and
  duplicates state that already lives in the local DB.
- **Glance 1.2/1.3 alphas**: newer features (responsive `SizeBox`, richer
  progress indicators) but pre-release; not worth the instability for v1.

## Consequences

- Widgets reflect local state immediately and converge with the server via the
  existing sync path — the same database-wins model as the app.
- Glance's UI subset applies: no arbitrary `Canvas`; progress uses Glance's
  `LinearProgressIndicator`; rows are capped in the widget and overflow shows
  "+n more in the app".
- Widget taps required a small navigation change: `MainActivity` now handles
  deep-link extras (`singleTop` + `onNewIntent`).
- The widget background/theme is a Glance `ColorProviders` mirror of the app's
  brand palette (light + dark), since Glance cannot reuse the Compose theme.

## When to Revisit

- When Glance 1.2+ reaches stable, evaluate `SizeMode.Responsive` (adaptive
  layouts per widget size) and `CircularProgressIndicator` for a habit ring.
- If more widget types appear (daily plan, focus timer), consider a per-widget
  configuration picker (choose a task list / habit set) via a configuration
  activity backed by the existing `GlanceStateDefinition` preferences.

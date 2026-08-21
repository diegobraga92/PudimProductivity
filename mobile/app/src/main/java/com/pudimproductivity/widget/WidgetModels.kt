package com.pudimproductivity.widget

import com.pudimproductivity.local.LocalCompletion
import com.pudimproductivity.local.LocalTask
import com.pudimproductivity.utils.computeStreaks
import java.time.LocalDate

/**
 * View-models and pure mapping functions for the Phase 10 home-screen widgets.
 *
 * The `build*` functions take plain local-DB entities (ADR 012) and return
 * small UI snapshots. They have no Android dependencies, so they can be
 * unit-tested on the JVM (see WidgetModelsTest.kt).
 */

/** One row in the "Today's Tasks" widget. */
data class TaskRow(val id: String, val title: String, val done: Boolean)

/** Full state for the "Today's Tasks" widget. */
data class TasksSnapshot(
    /**
     * All rows in display order: pending first, then done (alphabetical within
     * each group). Done rows stay visible so completed work reinforces
     * progress — they render with a restrained strikethrough.
     */
    val visible: List<TaskRow>,
    val done: Int,
    val total: Int
) {
    /** Tasks still to complete — what the widget header should emphasise. */
    val pending: List<TaskRow> get() = visible.filterNot { it.done }
    val remaining: Int get() = pending.size
}

/** One row in the "Today's Habits" widget. */
data class HabitRow(
    val id: String,
    val title: String,
    val completedToday: Boolean,
    val streak: Int,
    val bestStreak: Int
)

/** Full state for the "Today's Habits" widget. */
data class HabitsSnapshot(
    val habits: List<HabitRow>,
    val doneToday: Int,
    val scheduledToday: Int
)

/**
 * Pure mapping for the Tasks widget: only one-off tasks (no recurrence) are
 * shown — recurring habits live in the Habits widget. Rows are ordered
 * pending first, then done, each group alphabetically, so the widget can show
 * completed tasks without hiding the work still to do.
 */
fun buildTasksSnapshot(tasks: List<LocalTask>): TasksSnapshot {
    val rows = tasks
        .filter { !it.deleted && it.list_id == null && it.recurrence_days.isNullOrEmpty() }
        .sortedWith(compareBy({ it.status == "done" }, { it.title.lowercase() }))
        .map { TaskRow(id = it.id, title = it.title, done = it.status == "done") }

    return TasksSnapshot(
        visible = rows,
        done = rows.count { it.done },
        total = rows.size
    )
}

/**
 * Pure mapping for the Habits widget. A habit is shown when it is scheduled on
 * [todayDay] (e.g. "mon") or was already completed today, so an off-schedule
 * completion still surfaces with its box ticked. Streaks reuse the app's
 * [computeStreaks] logic (same as HabitScreen).
 */
fun buildHabitsSnapshot(
    tasks: List<LocalTask>,
    completions: List<LocalCompletion>,
    today: String,
    todayDay: String
): HabitsSnapshot {
    val completedTodayIds = completions
        .filter { !it.deleted && it.completed_date == today }
        .mapTo(mutableSetOf()) { it.task_id }

    val habits = tasks
        .filter { !it.deleted && it.list_id == null && !it.recurrence_days.isNullOrEmpty() }
        .filter { it.recurrence_days.orEmpty().contains(todayDay) || it.id in completedTodayIds }
        .sortedBy { it.title.lowercase() }
        .map { task ->
            val dates = completions
                .filter { !it.deleted && it.task_id == task.id }
                .map { it.completed_date }
            val streak = computeStreaks(dates)
            HabitRow(
                id = task.id,
                title = task.title,
                completedToday = task.id in completedTodayIds,
                streak = streak.current,
                bestStreak = streak.longest
            )
        }

    return HabitsSnapshot(
        habits = habits,
        doneToday = habits.count { it.completedToday },
        scheduledToday = habits.size
    )
}

/** Monday-first day key ("mon".."sun") for a date, mirroring the app's DAY_ORDER. */
fun dayName(date: LocalDate): String {
    val dayIndex = (date.dayOfWeek.value + 6) % 7 // 0 = Monday
    return DAY_ORDER[dayIndex]
}

internal val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")

package com.pudimproductivity.widget

import com.pudimproductivity.local.LocalCompletion
import com.pudimproductivity.local.LocalTask
import com.pudimproductivity.utils.computeStreaks
import java.time.LocalDate

/**
 * View-models and pure mapping functions for the home-screen widgets.
 *
 * The `build*` functions take plain local-DB entities and return
 * small UI snapshots. They have no Android dependencies, so they can be
 * unit-tested on the JVM.
 */

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

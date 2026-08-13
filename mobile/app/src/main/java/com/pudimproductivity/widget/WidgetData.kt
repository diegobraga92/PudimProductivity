package com.pudimproductivity.widget

import android.content.Context
import com.pudimproductivity.local.LocalDatabase
import java.time.LocalDate

/**
 * Loads widget state straight from the offline-first local SQLite DB — the
 * same source of truth the app UI uses (ADR 012). Widgets render instantly,
 * work offline, and never touch the network in the render path.
 */
object WidgetData {

    fun loadTasks(context: Context): TasksSnapshot =
        buildTasksSnapshot(tasks = LocalDatabase(context).queryTasks())

    fun loadHabits(context: Context): HabitsSnapshot {
        val today = LocalDate.now()
        val db = LocalDatabase(context)
        return buildHabitsSnapshot(
            tasks = db.queryTasks(),
            completions = db.queryCompletions(),
            today = today.toString(),
            todayDay = dayName(today)
        )
    }
}

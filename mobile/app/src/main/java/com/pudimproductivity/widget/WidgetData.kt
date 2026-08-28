package com.pudimproductivity.widget

import android.content.Context
import com.pudimproductivity.local.LocalDatabase
import java.time.LocalDate

/**
 * Loads widget state straight from the offline-first local SQLite DB.
 */
object WidgetData {

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

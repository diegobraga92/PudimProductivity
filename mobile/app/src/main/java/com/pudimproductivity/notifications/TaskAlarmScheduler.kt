package com.pudimproductivity.notifications

import android.content.Context
import androidx.work.ExistingWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.workDataOf
import com.pudimproductivity.local.LocalDatabase
import java.time.Duration
import java.time.LocalDate
import java.time.ZonedDateTime
import java.util.concurrent.TimeUnit

/**
 * Planner alarms: a habit with `start_time` and `alarm_minutes` fires a local
 * notification `alarm_minutes` before `start_time` on each of its recurrence
 * days, mirroring the web app's alarm semantics. The schedule is pulled into the local
 * DB by the sync worker, so alarms fire with the app closed and fully offline.
 */
object TaskAlarmScheduler {

    private const val WORK_PREFIX = "pudim.alarm."
    internal const val KEY_TASK_ID = "task_id"

    /**
     * Full reschedule: cancels every pending task alarm and re-arms the next
     * occurrence of each habit that has a planner alarm.
     */
    fun schedule(context: Context) {
        cancelAll(context)
        LocalDatabase(context).queryTasks()
            .filter { it.recurrence_days?.isNotEmpty() == true }
            .filter { !it.start_time.isNullOrBlank() }
            .filter { it.alarm_minutes != null && it.alarm_minutes > 0 }
            .forEach { scheduleTaskFor(context, it.id) }
    }

    fun rescheduleAll(context: Context) = schedule(context)

    /**
     * Arms the next alarm occurrence for a single task by reading its current
     * row from the local DB. Also used by [TaskAlarmWorker] after it fires to
     * re-arm the following occurrence of a recurring habit.
     */
    fun scheduleTaskFor(context: Context, taskId: String) {
        val task = LocalDatabase(context).queryTaskById(taskId) ?: return
        val alarmMinutes = task.alarm_minutes ?: return
        if (task.recurrence_days.isNullOrEmpty()) return
        val startTime = task.start_time ?: return

        val now = ZonedDateTime.now()
        val next = nextAlarmDateTime(startTime, alarmMinutes, task.recurrence_days, now) ?: return
        val delay = Duration.between(now, next)
        if (delay.isNegative || delay.isZero) return

        val request = OneTimeWorkRequestBuilder<TaskAlarmWorker>()
            .setInitialDelay(delay.toMillis(), TimeUnit.MILLISECONDS)
            .setInputData(workDataOf(KEY_TASK_ID to taskId))
            .build()
        WorkManager.getInstance(context)
            .enqueueUniqueWork(WORK_PREFIX + taskId, ExistingWorkPolicy.REPLACE, request)
    }

    /** Cancels pending alarms for every task the local DB knows, including tombstones. */
    fun cancelAll(context: Context) {
        val wm = WorkManager.getInstance(context)
        LocalDatabase(context).queryAllTaskIds().forEach { wm.cancelUniqueWork(WORK_PREFIX + it) }
    }

    /**
     * The next alarm datetime strictly after [now] for a habit that recurs on
     * [recurrenceDays] ("mon".."sun"). Returns null when the offset lands
     * before midnight or no future recurrence day exists.
     */
    internal fun nextAlarmDateTime(
        startTime: String,
        alarmMinutes: Int,
        recurrenceDays: List<String>,
        now: ZonedDateTime
    ): ZonedDateTime? {
        val startMinutes = parseTimeToMinutes(startTime) ?: return null
        val alarmMinutesOfDay = startMinutes - alarmMinutes
        if (alarmMinutesOfDay < 0) return null

        val days = recurrenceDays.map { it.lowercase() }
        // Habits recur weekly, so scanning the next 8 days always finds a
        // matching day for a valid recurrence set.
        for (offset in 0..7) {
            val date = now.toLocalDate().plusDays(offset.toLong())
            if (weekdayKey(date) in days) {
                val candidate = date
                    .atTime(alarmMinutesOfDay / 60, alarmMinutesOfDay % 60)
                    .atZone(now.zone)
                if (candidate.isAfter(now)) return candidate
            }
        }
        return null
    }

    private fun weekdayKey(date: LocalDate): String {
        // java.time: Monday = 1 → map to the API's "mon".."sun" keys.
        val keys = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
        return keys[date.dayOfWeek.value - 1]
    }

    /** Parses "HH:MM" into minutes-of-day, null for malformed input. */
    internal fun parseTimeToMinutes(t: String): Int? {
        val parts = t.split(":")
        val h = parts.getOrNull(0)?.toIntOrNull() ?: return null
        val m = parts.getOrNull(1)?.toIntOrNull() ?: 0
        if (h !in 0..23 || m !in 0..59) return null
        return h * 60 + m
    }
}

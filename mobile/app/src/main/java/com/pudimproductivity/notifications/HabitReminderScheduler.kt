package com.pudimproductivity.notifications

import android.content.Context
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.time.Duration
import java.time.ZonedDateTime
import java.util.concurrent.TimeUnit

/**
 * Schedules the daily local habit reminder. The initial run is aligned to the
 * next 08:00 (device-local time); WorkManager then repeats it every ~24h, so it
 * lands in the morning window. Runs deferred by battery optimizations still
 * post when they eventually execute (late beats never).
 */
object HabitReminderScheduler {

    private const val WORK_NAME = "pudim.habit-reminder"
    private const val REMINDER_HOUR = 8

    fun schedule(context: Context) {
        val request = PeriodicWorkRequestBuilder<HabitReminderWorker>(1, TimeUnit.DAYS)
            .setInitialDelay(millisUntilNext(REMINDER_HOUR), TimeUnit.MILLISECONDS)
            .build()
        WorkManager.getInstance(context)
            .enqueueUniquePeriodicWork(WORK_NAME, ExistingPeriodicWorkPolicy.KEEP, request)
    }

    /** Milliseconds from `now` until the next occurrence of `hour` (device-local time). */
    internal fun millisUntilNext(hour: Int, now: ZonedDateTime = ZonedDateTime.now()): Long {
        var next = now.withHour(hour).withMinute(0).withSecond(0).withNano(0)
        if (!next.isAfter(now)) next = next.plusDays(1)
        return Duration.between(now, next).toMillis()
    }
}

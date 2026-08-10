package com.pudimproductivity.notifications

import android.content.Context
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.time.Duration
import java.util.concurrent.TimeUnit

/**
 * Schedules the daily local habit reminder (every ~24h; the worker only posts
 * when it runs in the morning window).
 */
object HabitReminderScheduler {

    private const val WORK_NAME = "pudim.habit-reminder"

    fun schedule(context: Context) {
        val request = PeriodicWorkRequestBuilder<HabitReminderWorker>(1, TimeUnit.DAYS)
            .setInitialDelay(Duration.ofHours(8).toMillis(), TimeUnit.MILLISECONDS)
            .build()
        WorkManager.getInstance(context)
            .enqueueUniquePeriodicWork(WORK_NAME, ExistingPeriodicWorkPolicy.KEEP, request)
    }
}

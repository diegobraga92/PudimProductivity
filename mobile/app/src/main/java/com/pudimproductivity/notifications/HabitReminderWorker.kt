package com.pudimproductivity.notifications

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import androidx.work.Worker
import androidx.work.WorkerParameters
import com.pudimproductivity.local.LocalDatabase
import java.time.LocalDate
import java.time.format.DateTimeFormatter

/**
 * Phase 9c local habit reminder: every morning at 08:00 checks habits not yet
 * completed today (from the local DB) and posts a notification. Works fully
 * offline — no backend push dependency.
 */
class HabitReminderWorker(
    context: Context,
    params: WorkerParameters
) : Worker(context, params) {

    override fun doWork(): Result {
        val appContext = applicationContext
        val db = LocalDatabase(appContext)
        val today = LocalDate.now().format(DateTimeFormatter.ISO_DATE)

        val pendingHabits = db.queryTasks()
            .filter { !it.recurrence_days.isNullOrEmpty() }
            .filter { task ->
                val completed = db.queryCompletions()
                    .any { it.task_id == task.id && it.completed_date == today && !it.deleted }
                !completed
            }

        if (pendingHabits.isNotEmpty()) {
            val title = "🌅 Good morning!"
            val text = "${pendingHabits.size} habit${if (pendingHabits.size == 1) "" else "s"} not done yet today: " +
                pendingHabits.take(3).joinToString(", ") { it.title } +
                if (pendingHabits.size > 3) "…" else ""

            post(appContext, title, text)
        }
        return Result.success()
    }

    companion object {
        private const val CHANNEL_ID = "pudim.habits"
        private const val NOTIFICATION_ID = 1001

        private fun ensureChannel(context: Context) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Habit reminders",
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply { description = "Local daily reminders for habits not yet completed" }
            context.getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }

        private fun post(context: Context, title: String, text: String) {
            ensureChannel(context)
            if (ContextCompat.checkSelfPermission(context, android.Manifest.permission.POST_NOTIFICATIONS)
                != PackageManager.PERMISSION_GRANTED
            ) {
                return
            }
            val notification = NotificationCompat.Builder(context, CHANNEL_ID)
                .setSmallIcon(android.R.drawable.ic_popup_reminder)
                .setContentTitle(title)
                .setContentText(text)
                .setAutoCancel(true)
                .build()
            NotificationManagerCompat.from(context).notify(NOTIFICATION_ID, notification)
        }
    }
}

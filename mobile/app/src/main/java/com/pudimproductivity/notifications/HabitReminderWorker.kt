package com.pudimproductivity.notifications

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import androidx.work.Worker
import androidx.work.WorkerParameters
import com.pudimproductivity.i18n.Localization
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
        Localization.init(appContext)
        val db = LocalDatabase(appContext)
        val today = LocalDate.now().format(DateTimeFormatter.ISO_DATE)

        val pendingHabits = db.queryTasks()
            .filter { it.list_id == null && !it.recurrence_days.isNullOrEmpty() }
            .filter { task ->
                val completed = db.queryCompletions()
                    .any { it.task_id == task.id && it.completed_date == today && !it.deleted }
                !completed
            }

        if (pendingHabits.isNotEmpty()) {
            val title = Localization.text("notifications.goodMorning")
            val text = Localization.text("notifications.pendingHabits", "count" to pendingHabits.size) +
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
                Localization.text("notifications.channel.habits"),
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply { description = Localization.text("notifications.channel.habitsDesc") }
            context.getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }

        private fun post(context: Context, title: String, text: String) {
            ensureChannel(context)
            // Android 13+ requires the runtime POST_NOTIFICATIONS permission;
            // on older devices the user can still disable notifications in
            // Settings. Post only when notifications can actually be shown.
            val permissionGranted = Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
                ContextCompat.checkSelfPermission(context, android.Manifest.permission.POST_NOTIFICATIONS) ==
                PackageManager.PERMISSION_GRANTED
            if (!permissionGranted || !NotificationManagerCompat.from(context).areNotificationsEnabled()) {
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

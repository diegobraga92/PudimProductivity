package com.pudimproductivity.notifications

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.local.LocalDatabase

/**
 * Fires a planner alarm: posts a local notification for a habit whose
 * start_time minus alarm_minutes has arrived, then arms the next occurrence
 * (habits recur weekly). Scheduled by [TaskAlarmScheduler] via WorkManager, so
 * it fires even when the app is closed. The schedule lives in the local DB, so
 * it works fully offline.
 */
class TaskAlarmWorker(
    appContext: Context,
    params: WorkerParameters
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        val appContext = applicationContext
        val taskId = inputData.getString(TaskAlarmScheduler.KEY_TASK_ID) ?: return Result.success()
        val db = LocalDatabase(appContext)
        val task = db.queryTaskById(taskId) ?: return Result.success() // deleted or unknown

        Localization.init(appContext)
        val time = task.start_time?.take(5) ?: "--:--"
        val body = Localization.text("notifications.alarm.body", "time" to time)
        post(appContext, task.id, task.title, body)

        // Habits recur weekly — arm the following occurrence.
        TaskAlarmScheduler.scheduleTaskFor(appContext, taskId)
        return Result.success()
    }

    companion object {
        private const val CHANNEL_ID = "pudim.alarms"

        private fun ensureChannel(context: Context) {
            Localization.init(context)
            val channel = NotificationChannel(
                CHANNEL_ID,
                Localization.text("notifications.channel.alarm"),
                NotificationManager.IMPORTANCE_DEFAULT
            ).apply {
                description = Localization.text("notifications.channel.alarmDesc")
            }
            context.getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }

        private fun post(context: Context, taskId: String, title: String, text: String) {
            ensureChannel(context)
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
            // Per-task id so simultaneous alarms don't overwrite each other.
            val notificationId = (taskId.hashCode() and 0x7fffffff).coerceAtLeast(1)
            NotificationManagerCompat.from(context).notify(notificationId, notification)
        }
    }
}

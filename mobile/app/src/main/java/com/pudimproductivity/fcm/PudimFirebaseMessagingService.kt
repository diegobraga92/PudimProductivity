package com.pudimproductivity.fcm

import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.pudimproductivity.R
import com.pudimproductivity.i18n.Localization

/**
 * Firebase Cloud Messaging handler (Phase 3). Receives push notifications from
 * the backend notifications worker and surfaces them as system notifications.
 *
 * Requires a real Firebase project: add `google-services.json` to
 * `mobile/app/` and the google-services Gradle plugin. Without it, this service
 * is simply never invoked.
 */
class PudimFirebaseMessagingService : FirebaseMessagingService() {

    override fun onMessageReceived(message: RemoteMessage) {
        val title = message.notification?.title ?: message.data["title"] ?: "Pudim"
        val body = message.notification?.body ?: message.data["body"] ?: ""

        createChannel()
        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .build()

        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(NOTIFICATION_ID++, notification)
    }

    override fun onNewToken(token: String) {
        // A real deployment would POST this token to the backend device
        // registry (Phase 8). For the single-user MVP, set it as the backend's
        // FCM_DEVICE_TOKEN env var.
        Log.d(TAG, "FCM device token: $token")
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Localization.init(this)
            val channel = NotificationChannel(
                CHANNEL_ID,
                Localization.text("notifications.channel.task"),
                NotificationManager.IMPORTANCE_DEFAULT
            )
            val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            manager.createNotificationChannel(channel)
        }
    }

    companion object {
        private const val TAG = "PudimFCM"
        private const val CHANNEL_ID = "pudim_task_notifications"
        private var NOTIFICATION_ID = 1001
    }
}

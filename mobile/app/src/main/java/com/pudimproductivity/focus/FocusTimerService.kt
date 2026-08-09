package com.pudimproductivity.focus

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import com.pudimproductivity.MainActivity
import com.pudimproductivity.R
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch

/**
 * Foreground service that keeps the focus timer alive when the app is
 * backgrounded and shows a persistent notification with the remaining time and
 * a Stop action. The timer state lives in [FocusTimerManager]; this service
 * renders it and only exists while a session is active.
 */
class FocusTimerService : Service() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private var isForeground = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            FocusTimerManager.ACTION_STOP -> {
                FocusTimerManager.stop()
                stopSelf()
                return START_NOT_STICKY
            }
            else -> startAsForeground()
        }
        return START_NOT_STICKY
    }

    private fun startAsForeground() {
        if (isForeground) return
        createChannel()

        val stopIntent = PendingIntent.getService(
            this, 0,
            Intent(this, FocusTimerService::class.java).setAction(FocusTimerManager.ACTION_STOP),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val contentIntent = PendingIntent.getActivity(
            this, 0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        startForeground(
            NOTIFICATION_ID,
            buildNotification("Focus timer", "Preparing…", stopIntent, contentIntent)
        )
        isForeground = true

        // Render the shared timer state into the notification.
        scope.launch {
            FocusTimerManager.state.collectLatest { state ->
                val remaining = formatTime(state.remainingSeconds)
                val status = when {
                    !state.active -> "Idle"
                    state.running -> "Focus — $remaining"
                    state.paused -> "Paused — $remaining"
                    else -> "Idle"
                }
                val nm = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
                nm.notify(NOTIFICATION_ID, buildNotification("Focus timer", status, stopIntent, contentIntent))
                if (!state.active) {
                    stopSelf()
                }
            }
        }
    }

    private fun buildNotification(
        title: String,
        text: String,
        stopIntent: PendingIntent,
        contentIntent: PendingIntent
    ): Notification = NotificationCompat.Builder(this, CHANNEL_ID)
        .setSmallIcon(R.drawable.ic_launcher_foreground)
        .setContentTitle(title)
        .setContentText(text)
        .setContentIntent(contentIntent)
        .setOngoing(true)
        .addAction(0, "Stop", stopIntent)
        .build()

    private fun createChannel() {
        // minSdk = 26, so NotificationChannel is always available.
        val channel = NotificationChannel(
            CHANNEL_ID,
            "Focus timer",
            NotificationManager.IMPORTANCE_LOW
        )
        (getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager)
            .createNotificationChannel(channel)
    }

    override fun onDestroy() {
        super.onDestroy()
        scope.cancel()
    }

    private fun formatTime(totalSeconds: Int): String {
        val m = totalSeconds / 60
        val s = totalSeconds % 60
        return "%02d:%02d".format(m, s)
    }

    companion object {
        private const val CHANNEL_ID = "pudim_focus_timer"
        private const val NOTIFICATION_ID = 2001
    }
}

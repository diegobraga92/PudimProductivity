package com.pudimproductivity.sync

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.pudimproductivity.widget.WidgetUpdater

/**
 * Background sync worker. Scheduled periodically and on app
 * foreground; pushes local dirty rows and pulls incremental changes.
 */
class SyncWorker(
    appContext: Context,
    params: WorkerParameters
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            SyncManager(applicationContext).sync()
            // Widgets read the local DB directly, so refresh them
            // after the server state is applied.
            WidgetUpdater.updateAll(applicationContext)
            Result.success()
        } catch (_: Exception) {
            Result.retry()
        }
    }
}

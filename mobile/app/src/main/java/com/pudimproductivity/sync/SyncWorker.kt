package com.pudimproductivity.sync

import android.content.Context
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters

/**
 * Phase 9c background sync worker. Scheduled periodically and on app
 * foreground; pushes local dirty rows and pulls incremental changes.
 */
class SyncWorker(
    appContext: Context,
    params: WorkerParameters
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        return try {
            SyncManager(applicationContext).sync()
            Result.success()
        } catch (_: Exception) {
            Result.retry()
        }
    }
}

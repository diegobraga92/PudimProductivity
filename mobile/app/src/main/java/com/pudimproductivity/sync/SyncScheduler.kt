package com.pudimproductivity.sync

import android.content.Context
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

/**
 * Schedules Phase 9c background sync:
 *  - an immediate one-time sync on app start,
 *  - a periodic sync every 15 minutes while connected to the network.
 */
object SyncScheduler {

    private const val PERIODIC_WORK = "pudim.periodic-sync"
    private const val INITIAL_WORK = "pudim.initial-sync"

    fun schedule(context: Context) {
        val wm = WorkManager.getInstance(context)

        val networkConstraint = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        wm.enqueueUniqueWork(
            INITIAL_WORK,
            ExistingWorkPolicy.REPLACE,
            OneTimeWorkRequestBuilder<SyncWorker>()
                .setConstraints(networkConstraint)
                .build()
        )

        val periodic = PeriodicWorkRequestBuilder<SyncWorker>(15, TimeUnit.MINUTES)
            .setConstraints(networkConstraint)
            .build()
        wm.enqueueUniquePeriodicWork(PERIODIC_WORK, ExistingPeriodicWorkPolicy.KEEP, periodic)
    }
}

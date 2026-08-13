package com.pudimproductivity.widget

import android.content.Context
import androidx.glance.appwidget.updateAll

/**
 * Refreshes every placed widget. Glance has no reactive update channel, so the
 * app pushes fresh data explicitly after local writes and after background
 * syncs (see TaskRepository.refreshFromLocal and SyncWorker).
 */
object WidgetUpdater {

    suspend fun updateAll(context: Context) {
        TasksWidget.updateAll(context)
        HabitsWidget.updateAll(context)
    }
}

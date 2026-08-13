package com.pudimproductivity.widget

import android.content.Context
import androidx.glance.GlanceId
import androidx.glance.action.ActionParameters
import androidx.glance.action.actionParametersOf
import androidx.glance.appwidget.action.ActionCallback
import androidx.glance.appwidget.action.actionRunCallback
import androidx.work.ExistingWorkPolicy
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import com.pudimproductivity.MainActivity
import com.pudimproductivity.local.LocalCompletion
import com.pudimproductivity.local.LocalDatabase
import com.pudimproductivity.sync.SyncWorker
import java.time.Instant
import java.time.LocalDate
import java.util.UUID
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Parameter keys shared between the widget composables and
 * [WidgetActionCallback] (simple types only — Glance parcels them with the
 * broadcast intent).
 */
internal val KEY_ACTION = ActionParameters.Key<String>("action")
internal val KEY_TASK_ID = ActionParameters.Key<String>("task_id")
internal val KEY_DONE = ActionParameters.Key<Boolean>("done")

internal const val ACTION_TOGGLE_TASK = "toggle_task"
internal const val ACTION_TOGGLE_HABIT = "toggle_habit"

/**
 * Deep-link keys for `actionStartActivity` → [MainActivity]. Named after the
 * intent extras the activity reads (`EXTRA_SCREEN` / `EXTRA_TASK_ID`); Glance
 * writes the parameters into the launched intent as typed extras.
 */
internal val EXTRA_SCREEN_KEY = ActionParameters.Key<String>(MainActivity.EXTRA_SCREEN)
internal val EXTRA_TASK_ID_KEY = ActionParameters.Key<String>(MainActivity.EXTRA_TASK_ID)

/** Checkbox action: mark a one-off task done / not done. */
internal fun toggleTaskAction(taskId: String, done: Boolean) =
    actionRunCallback<WidgetActionCallback>(
        actionParametersOf(
            KEY_ACTION to ACTION_TOGGLE_TASK,
            KEY_TASK_ID to taskId,
            KEY_DONE to done
        )
    )

/** Checkbox action: complete / uncomplete a habit for today. */
internal fun toggleHabitAction(taskId: String) =
    actionRunCallback<WidgetActionCallback>(
        actionParametersOf(
            KEY_ACTION to ACTION_TOGGLE_HABIT,
            KEY_TASK_ID to taskId
        )
    )

/**
 * Broadcast-receiver callback fired when a widget checkbox is tapped.
 *
 * Writes the change optimistically to the local DB with the same dirty-flag
 * semantics as `TaskRepository` (ADR 012), then enqueues a background sync so
 * the server converges. Glance invokes [onAction] in a coroutine off the main
 * thread; the DB writes are additionally dispatched to [Dispatchers.IO].
 */
class WidgetActionCallback : ActionCallback {

    override suspend fun onAction(
        context: Context,
        glanceId: GlanceId,
        parameters: ActionParameters
    ) {
        when (parameters[KEY_ACTION]) {
            ACTION_TOGGLE_TASK -> {
                val taskId = parameters[KEY_TASK_ID] ?: return
                val done = parameters[KEY_DONE] ?: return
                toggleTask(context, taskId, done)
            }
            ACTION_TOGGLE_HABIT -> {
                val taskId = parameters[KEY_TASK_ID] ?: return
                toggleHabit(context, taskId)
            }
            else -> return
        }
        // Re-render both widgets — either dataset may have changed.
        TasksWidget.update(context, glanceId)
        HabitsWidget.update(context, glanceId)
    }

    private suspend fun toggleTask(context: Context, taskId: String, done: Boolean) {
        withContext(Dispatchers.IO) {
            val db = LocalDatabase(context)
            val existing = db.queryTaskById(taskId) ?: return@withContext
            db.upsertTasks(
                listOf(
                    existing.copy(
                        status = if (done) "done" else "todo",
                        updated_at = Instant.now().toString(),
                        dirty = true
                    )
                )
            )
        }
        enqueueSync(context)
    }

    private suspend fun toggleHabit(context: Context, taskId: String) {
        withContext(Dispatchers.IO) {
            val db = LocalDatabase(context)
            val today = LocalDate.now().toString()
            val existing = db.queryCompletions().firstOrNull {
                it.task_id == taskId && it.completed_date == today && !it.deleted
            }
            if (existing != null) {
                db.markCompletionDeleted(existing.id)
            } else {
                db.upsertCompletions(
                    listOf(
                        LocalCompletion(
                            id = UUID.randomUUID().toString(),
                            task_id = taskId,
                            completed_date = today,
                            created_at = Instant.now().toString(),
                            dirty = true
                        )
                    )
                )
            }
        }
        enqueueSync(context)
    }

    /** Fire-and-forget background sync so the optimistic write reaches the server. */
    private fun enqueueSync(context: Context) {
        WorkManager.getInstance(context).enqueueUniqueWork(
            "pudim.widget-sync",
            ExistingWorkPolicy.REPLACE,
            OneTimeWorkRequestBuilder<SyncWorker>().build()
        )
    }
}

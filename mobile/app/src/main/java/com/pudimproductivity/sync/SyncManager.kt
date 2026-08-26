package com.pudimproductivity.sync

import android.content.Context
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateTaskRequest
import com.pudimproductivity.api.UpdateTaskRequest
import com.pudimproductivity.api.syncService
import com.pudimproductivity.api.taskService
import com.pudimproductivity.local.LocalCompletion
import com.pudimproductivity.local.LocalDatabase
import com.pudimproductivity.local.LocalShare
import com.pudimproductivity.local.LocalTask
import com.pudimproductivity.local.LocalTaskList
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext

/**
 * Phase 9c offline sync:
 *
 *  - **push** local dirty rows to the server via the existing REST endpoints
 *    (create/update/delete/complete). On success the rows are marked clean.
 *    HTTP 409 from a merge (Phase 8 LWW) is treated as a conflict: the server
 *    state is pulled in the next pass and the local dirty flag is cleared.
 *  - **pull** incremental changes from `GET /api/v1/sync?since=...` and apply
 *    them to the local SQLite database (upsert active rows, tombstone deleted).
 *
 * Run from a WorkManager worker and on app foreground/connectivity restore.
 */
class SyncManager(private val context: Context) {

    companion object {
        // Serializes syncs across every SyncManager instance — the app repository,
        // the WorkManager worker and the reconnect hooks all create their own —
        // so push/pull never run concurrently against the same SQLite database.
        private val syncMutex = Mutex()
    }

    private val db by lazy { LocalDatabase(context) }

    /** Full sync: push local changes, then pull the server's. */
    suspend fun sync() {
        syncMutex.withLock {
            withContext(Dispatchers.IO) {
                pushLocalChanges()
                pullChangesLocked()
            }
        }
    }

    /** Only fetch server changes (after connect or WS event). */
    suspend fun pullChanges() {
        syncMutex.withLock { pullChangesLocked() }
    }

    private suspend fun pullChangesLocked() {
        withContext(Dispatchers.IO) {
            val since = db.getLastSyncTs()
            val bundle = ApiClient.syncService.getChanges(since)
            if (bundle.timestamp.isBlank()) return@withContext

            db.upsertTasks(bundle.tasks.map { t ->
                LocalTask(
                    id = t.id, title = t.title, status = t.status,
                    recurrence_days = t.recurrence_days, list_id = t.list_id,
                    // Planner scheduling fields — needed locally so the alarm
                    // scheduler (start_time − alarm_minutes) works offline.
                    start_time = t.start_time, end_time = t.end_time,
                    color = t.color, scheduled_date = t.scheduled_date,
                    alarm_minutes = t.alarm_minutes,
                    created_at = t.created_at, updated_at = t.updated_at,
                    // Server rows exist remotely, so a later local edit is an UPDATE.
                    synced = true
                )
            })
            db.upsertCompletions(bundle.completions.map { c ->
                LocalCompletion(
                    id = c.id, task_id = c.task_id,
                    completed_date = c.completed_date, created_at = c.created_at
                )
            })
            db.upsertTaskLists(bundle.task_lists.map { l ->
                LocalTaskList(
                    id = l.id, name = l.name, description = l.description,
                    owner_id = l.owner_id, created_at = l.created_at, updated_at = l.updated_at,
                    synced = true
                )
            })
            db.upsertShares(bundle.shares.map { s ->
                LocalShare(list_id = s.list_id, shared_with = s.shared_with, role = s.role, created_at = s.created_at)
            })

            db.applyDeletedTaskIds(bundle.deleted_task_ids)
            db.applyDeletedCompletionIds(bundle.deleted_completion_ids)
            db.applyDeletedTaskListIds(bundle.deleted_task_list_ids)
            bundle.deleted_share_keys.forEach { db.markShareDeleted(it) }

            db.setLastSyncTs(bundle.timestamp)
        }
    }


    /** Push locally created/updated/deleted rows; clear their dirty flags. */
    private suspend fun pushLocalChanges() {
        val api = ApiClient.taskService

        // Deleted tasks first (tombstones). A tombstone whose id was never
        // created on the server (offline create then delete) is just dropped
        // locally — there is nothing to delete remotely.
        db.dirtyTasks().filter { it.deleted }.forEach { t ->
            try {
                if (t.synced) {
                    api.deleteTask(t.id)
                    db.markTaskDirty(t.id, dirty = false)
                } else {
                    db.deleteLocalTask(t.id)
                }
            } catch (_: Exception) {
                // Retry next sync.
            }
        }
        // Created/updated tasks. `synced` (not a timestamp comparison) decides
        // whether the row already exists on the server: an offline-created task
        // that was also edited before its first push must CREATE first, then
        // re-apply the full local state (create can't carry status).
        db.dirtyTasks().filter { !it.deleted }.forEach { t ->
            try {
                if (t.synced) {
                    api.updateTask(
                        t.id,
                        UpdateTaskRequest(
                            title = t.title,
                            status = t.status,
                            recurrence_days = t.recurrence_days,
                            list_id = t.list_id
                        )
                    )
                } else {
                    api.createTask(
                        CreateTaskRequest(
                            id = t.id,
                            title = t.title,
                            recurrence_days = t.recurrence_days,
                            list_id = t.list_id
                        )
                    )
                    // Re-apply the full local state (status, etc.) so edits made
                    // offline before the first push are not lost.
                    api.updateTask(
                        t.id,
                        UpdateTaskRequest(
                            title = t.title,
                            status = t.status,
                            recurrence_days = t.recurrence_days,
                            list_id = t.list_id
                        )
                    )
                }
                db.markTaskSynced(t.id)
                db.markTaskDirty(t.id, dirty = false)
            } catch (_: Exception) {
                // Offline or conflict — keep dirty and retry; the pull pass will
                // converge any server-side winner.
            }
        }

        // Dirty completions: un-completed locally = DELETE; added = POST complete.
        // Tombstones are included so a local uncomplete actually reaches the
        // server; a successful push marks the row clean (the pull pass then
        // hard-deletes confirmed tombstones via deleted_completion_ids).
        db.dirtyCompletions().forEach { c ->
            try {
                if (c.deleted) {
                    api.uncompleteTask(c.task_id, c.completed_date)
                } else {
                    api.completeTask(c.task_id, c.completed_date, c.id)
                }
                db.markCompletionClean(c.id)
            } catch (_: Exception) {
                // Retry next sync.
            }
        }

        // Dirty task lists (same create-then-apply pattern as tasks).
        db.queryTaskLists().filter { it.dirty }.forEach { l ->
            try {
                if (l.deleted) {
                    if (l.synced) {
                        api.deleteTaskList(l.id)
                        // Clean the tombstone so the pull pass hard-deletes it
                        // (mirrors the tasks path).
                        db.markTaskListDirty(l.id, dirty = false)
                    } else {
                        db.deleteLocalTaskList(l.id)
                    }
                } else if (l.synced) {
                    api.updateTaskList(l.id, com.pudimproductivity.api.UpdateTaskListRequest(name = l.name, description = l.description))
                    db.markTaskListDirty(l.id, dirty = false)
                } else {
                    api.createTaskList(com.pudimproductivity.api.CreateTaskListRequest(id = l.id, name = l.name))
                    api.updateTaskList(l.id, com.pudimproductivity.api.UpdateTaskListRequest(name = l.name, description = l.description))
                    db.markTaskListSynced(l.id)
                    db.markTaskListDirty(l.id, dirty = false)
                }
            } catch (_: Exception) {
                // Retry next sync.
            }
        }
    }
}

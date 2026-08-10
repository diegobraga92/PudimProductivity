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

    private val db by lazy { LocalDatabase(context) }

    /** Full sync: push local changes, then pull the server's. */
    suspend fun sync() {
        withContext(Dispatchers.IO) {
            pushLocalChanges()
            pullChanges()
        }
    }

    /** Only fetch server changes (after connect or WS event). */
    suspend fun pullChanges() {
        withContext(Dispatchers.IO) {
            val since = db.getLastSyncTs()
            val bundle = ApiClient.syncService.getChanges(since)
            if (bundle.timestamp.isBlank()) return@withContext

            db.upsertTasks(bundle.tasks.map { t ->
                LocalTask(
                    id = t.id, title = t.title, status = t.status,
                    recurrence_days = t.recurrence_days, list_id = t.list_id,
                    created_at = t.created_at, updated_at = t.updated_at
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
                    owner_id = l.owner_id, created_at = l.created_at, updated_at = l.updated_at
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

        // Deleted tasks first (tombstones).
        db.dirtyTasks().filter { it.deleted }.forEach { t ->
            try {
                api.deleteTask(t.id)
                db.markTaskDirty(t.id, dirty = false)
            } catch (_: Exception) {
                // Retry next sync.
            }
        }
        // Created/updated tasks.
        db.dirtyTasks().filter { !it.deleted }.forEach { t ->
            try {
                val existing = db.queryTaskById(t.id)
                if (existing == null || existing.updated_at == t.created_at) {
                    api.createTask(
                        CreateTaskRequest(
                            title = t.title,
                            recurrence_days = t.recurrence_days,
                            list_id = t.list_id
                        )
                    )
                } else {
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
                db.markTaskDirty(t.id, dirty = false)
            } catch (_: Exception) {
                // Offline or conflict — keep dirty and retry; the pull pass will
                // converge any server-side winner.
            }
        }

        // Dirty completions: un-completed locally = DELETE; added = POST complete.
        db.queryCompletions().filter { it.dirty }.forEach { c ->
            try {
                if (c.deleted) {
                    api.uncompleteTask(c.task_id, c.completed_date)
                } else {
                    api.completeTask(c.task_id, c.completed_date)
                }
                if (c.deleted) db.markCompletionDeleted(c.id) else db.markCompletionClean(c.id)
            } catch (_: Exception) {
                // Retry next sync.
            }
        }

        // Dirty task lists.
        db.queryTaskLists().filter { it.dirty }.forEach { l ->
            try {
                if (l.deleted) {
                    api.deleteTaskList(l.id)
                    db.markTaskListDirty(l.id)
                } else {
                    val existing = db.queryTaskLists().firstOrNull { it.id == l.id }
                    if (existing?.updated_at == l.created_at) {
                        api.createTaskList(com.pudimproductivity.api.CreateTaskListRequest(l.name))
                    } else {
                        api.updateTaskList(l.id, com.pudimproductivity.api.UpdateTaskListRequest(name = l.name, description = l.description))
                    }
                    db.markTaskListDirty(l.id, dirty = false)
                }
            } catch (_: Exception) {
                // Retry next sync.
            }
        }
    }
}

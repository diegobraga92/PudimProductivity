package com.pudimproductivity.data

import android.content.Context
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.TaskCompletion
import com.pudimproductivity.api.TaskList
import com.pudimproductivity.local.LocalCompletion
import com.pudimproductivity.local.LocalDatabase
import com.pudimproductivity.local.LocalTask
import com.pudimproductivity.local.LocalTaskList
import com.pudimproductivity.sync.SyncManager
import com.pudimproductivity.widget.WidgetUpdater
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * Phase 9c local-first repository. All reads come from the local SQLite
 * database (instant, works offline); writes are optimistic (row inserted with
 * `dirty=1`) and flushed to the server by [SyncManager] on the next sync.
 */
class TaskRepository(private val context: Context, scope: CoroutineScope) {

    private val db = LocalDatabase(context)
    private val sync = SyncManager(context)
    private val appScope = scope

    private val _tasks = MutableStateFlow<List<Task>>(emptyList())
    val tasks: StateFlow<List<Task>> = _tasks.asStateFlow()

    private val _completions = MutableStateFlow<List<TaskCompletion>>(emptyList())
    val completions: StateFlow<List<TaskCompletion>> = _completions.asStateFlow()

    private val _taskLists = MutableStateFlow<List<TaskList>>(emptyList())
    val taskLists: StateFlow<List<TaskList>> = _taskLists.asStateFlow()

    private val _online = MutableStateFlow(true)
    val online: StateFlow<Boolean> = _online.asStateFlow()

    /** Loads the local snapshot into the flows, then attempts a background sync. */
    fun start() {
        refreshFromLocal()
        appScope.launch {
            try {
                sync.sync()
                refreshFromLocal()
                _online.value = true
            } catch (_: Exception) {
                _online.value = false
            }
        }
    }

    fun refreshFromLocal() {
        _tasks.value = db.queryTasks().map { it.toApi() }
        _completions.value = db.queryCompletions().map { it.toApi() }
        _taskLists.value = db.queryTaskLists().map { it.toApi() }
        // Phase 10: keep home-screen widgets in sync with the local snapshot.
        // Covers every local write, WS-event refresh, and post-sync re-emit.
        appScope.launch { WidgetUpdater.updateAll(context) }
    }

    // --- writes (optimistic, local-first) ---

    fun createTask(title: String, recurrenceDays: List<String>? = null, listId: String? = null) {
        val now = java.time.Instant.now().toString()
        val local = LocalTask(
            id = java.util.UUID.randomUUID().toString(),
            title = title,
            status = "todo",
            recurrence_days = recurrenceDays,
            list_id = listId,
            created_at = now,
            updated_at = now,
            dirty = true
        )
        withContextSafe { db.upsertTasks(listOf(local)) }
        refreshFromLocal()
        triggerSync()
    }

    fun updateTask(id: String, title: String? = null, status: String? = null) {
        val existing = db.queryTaskById(id) ?: return
        val updated = existing.copy(
            title = title ?: existing.title,
            status = status ?: existing.status,
            updated_at = java.time.Instant.now().toString(),
            dirty = true
        )
        withContextSafe { db.upsertTasks(listOf(updated)) }
        refreshFromLocal()
        triggerSync()
    }

    fun deleteTask(id: String) {
        withContextSafe { db.markTaskDeleted(id) }
        refreshFromLocal()
        triggerSync()
    }

    fun createTaskList(name: String) {
        val now = java.time.Instant.now().toString()
        val local = LocalTaskList(
            id = java.util.UUID.randomUUID().toString(),
            name = name,
            created_at = now,
            updated_at = now,
            dirty = true
        )
        withContextSafe { db.upsertTaskLists(listOf(local)) }
        refreshFromLocal()
        triggerSync()
    }

    fun deleteTaskList(id: String) {
        withContextSafe { db.markTaskListDeleted(id) }
        refreshFromLocal()
        triggerSync()
    }



    /** Record a local habit completion optimistically. */
    fun completeHabit(taskId: String, date: String) {
        val now = java.time.Instant.now().toString()
        val local = LocalCompletion(
            id = java.util.UUID.randomUUID().toString(),
            task_id = taskId,
            completed_date = date,
            created_at = now,
            dirty = true
        )
        withContextSafe { db.upsertCompletions(listOf(local)) }
        refreshFromLocal()
        triggerSync()
    }

    fun uncompleteHabit(taskId: String, date: String) {
        val existing = db.queryCompletions().firstOrNull { it.task_id == taskId && it.completed_date == date }
        if (existing != null) {
            withContextSafe { db.markCompletionDeleted(existing.id) }
        }
        refreshFromLocal()
        triggerSync()
    }

    fun setOnline(value: Boolean) {
        _online.value = value
    }

    /** Fire-and-forget server flush; pulls + re-emits on success. */
    private fun triggerSync() {
        appScope.launch {
            try {
                sync.sync()
                refreshFromLocal()
                _online.value = true
            } catch (_: Exception) {
                _online.value = false
            }
        }
    }

    private fun withContextSafe(block: () -> Unit) {
        kotlinx.coroutines.runBlocking {
            withContext(Dispatchers.IO) { block() }
        }
    }
}

private fun LocalTask.toApi(): Task = Task(
    id = id,
    title = title,
    status = status,
    recurrence_days = recurrence_days,
    list_id = list_id,
    created_at = created_at,
    updated_at = updated_at
)

private fun LocalCompletion.toApi(): TaskCompletion = TaskCompletion(
    id = id,
    task_id = task_id,
    completed_date = completed_date,
    created_at = created_at
)

private fun LocalTaskList.toApi(): TaskList = TaskList(
    id = id,
    name = name,
    description = description,
    owner_id = owner_id,
    created_at = created_at,
    updated_at = updated_at
)

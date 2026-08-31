package com.pudimproductivity.local

import android.content.ContentValues
import android.content.Context
import android.database.Cursor
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper

/**
 * Local SQLite database for offline-first operation.
 * Every table carries `dirty` (local change awaiting push) and `deleted`
 * (local or server tombstone) so the SyncManager can converge with the backend.
 */
class LocalDatabase(context: Context) : SQLiteOpenHelper(context.applicationContext, DB_NAME, null, DB_VERSION) {

    companion object {
        const val DB_NAME = "pudim_offline.db"
        const val DB_VERSION = 3

        const val META_KEY_LAST_SYNC = "last_sync_ts"
    }

    override fun onCreate(db: SQLiteDatabase) {
        db.execSQL("CREATE TABLE IF NOT EXISTS tasks (id TEXT PRIMARY KEY, title TEXT NOT NULL, status TEXT NOT NULL, recurrence_days TEXT, list_id TEXT, start_time TEXT, end_time TEXT, color TEXT, scheduled_date TEXT, alarm_minutes INTEGER, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, dirty INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0, synced INTEGER NOT NULL DEFAULT 0)")
        db.execSQL("CREATE TABLE IF NOT EXISTS completions (id TEXT PRIMARY KEY, task_id TEXT NOT NULL, completed_date TEXT NOT NULL, created_at TEXT NOT NULL, dirty INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0)")
        db.execSQL("CREATE TABLE IF NOT EXISTS task_lists (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', owner_id TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, dirty INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0, synced INTEGER NOT NULL DEFAULT 0)")
        db.execSQL("CREATE TABLE IF NOT EXISTS shares (list_id TEXT NOT NULL, shared_with TEXT NOT NULL, role TEXT NOT NULL, created_at TEXT NOT NULL, deleted INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (list_id, shared_with))")
        db.execSQL("CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)")
        db.execSQL("CREATE INDEX IF NOT EXISTS idx_tasks_deleted ON tasks (deleted)")
        db.execSQL("CREATE INDEX IF NOT EXISTS idx_completions_deleted ON completions (deleted)")
        db.execSQL("CREATE INDEX IF NOT EXISTS idx_task_lists_deleted ON task_lists (deleted)")
    }

    override fun onUpgrade(db: SQLiteDatabase, oldVersion: Int, newVersion: Int) {
        if (oldVersion < 2) {
            db.execSQL("ALTER TABLE tasks ADD COLUMN synced INTEGER NOT NULL DEFAULT 0")
            db.execSQL("ALTER TABLE task_lists ADD COLUMN synced INTEGER NOT NULL DEFAULT 0")
            db.execSQL("UPDATE tasks SET synced = 1")
            db.execSQL("UPDATE task_lists SET synced = 1")
        }
        if (oldVersion < 3) {
            db.execSQL("ALTER TABLE tasks ADD COLUMN start_time TEXT")
            db.execSQL("ALTER TABLE tasks ADD COLUMN end_time TEXT")
            db.execSQL("ALTER TABLE tasks ADD COLUMN color TEXT")
            db.execSQL("ALTER TABLE tasks ADD COLUMN scheduled_date TEXT")
            db.execSQL("ALTER TABLE tasks ADD COLUMN alarm_minutes INTEGER")
        }
    }

    // --- meta ---

    fun getLastSyncTs(): String? {
        val db = readableDatabase
        db.query("meta", arrayOf("value"), "key = ?", arrayOf(META_KEY_LAST_SYNC), null, null, null)
            .use { c ->
                if (c.moveToFirst()) return c.getString(0)
            }
        return null
    }

    fun setLastSyncTs(ts: String) {
        val db = writableDatabase
        db.insertWithOnConflict("meta", null, ContentValues().apply {
            put("key", META_KEY_LAST_SYNC)
            put("value", ts)
        }, SQLiteDatabase.CONFLICT_REPLACE)
    }

    // --- tasks ---

    fun upsertTasks(tasks: List<LocalTask>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (t in tasks) {
                val cv = ContentValues().apply {
                    put("id", t.id)
                    put("title", t.title)
                    put("status", t.status)
                    put("recurrence_days", encodeDays(t.recurrence_days))
                    put("list_id", t.list_id)
                    put("start_time", t.start_time)
                    put("end_time", t.end_time)
                    put("color", t.color)
                    put("scheduled_date", t.scheduled_date)
                    put("alarm_minutes", t.alarm_minutes)
                    put("created_at", t.created_at)
                    put("updated_at", t.updated_at)
                    put("dirty", if (t.dirty) 1 else 0)
                    put("deleted", if (t.deleted) 1 else 0)
                    put("synced", if (t.synced) 1 else 0)
                }
                db.insertWithOnConflict("tasks", null, cv, SQLiteDatabase.CONFLICT_REPLACE)
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun queryTasks(): List<LocalTask> {
        val db = readableDatabase
        val out = mutableListOf<LocalTask>()
        db.query("tasks", null, "deleted = 0", null, null, null, "created_at DESC").use { c ->
            while (c.moveToNext()) out.add(readTask(c))
        }
        return out
    }

    fun queryTaskById(id: String): LocalTask? {
        val db = readableDatabase
        db.query("tasks", null, "id = ? AND deleted = 0", arrayOf(id), null, null, null).use { c ->
            return if (c.moveToFirst()) readTask(c) else null
        }
    }

    fun queryTaskIncludingDeleted(id: String): LocalTask? {
        val db = readableDatabase
        db.query("tasks", null, "id = ?", arrayOf(id), null, null, null).use { c ->
            return if (c.moveToFirst()) readTask(c) else null
        }
    }

    fun markTaskDirty(id: String, dirty: Boolean = true) {
        writableDatabase.update("tasks", ContentValues().apply { put("dirty", if (dirty) 1 else 0) }, "id = ?", arrayOf(id))
    }

    /** Marks a task as confirmed on the server (a later edit should UPDATE). */
    fun markTaskSynced(id: String) {
        writableDatabase.update("tasks", ContentValues().apply { put("synced", 1) }, "id = ?", arrayOf(id))
    }

    /** Hard-deletes a local row (used for never-synced tombstones). */
    fun deleteLocalTask(id: String) {
        writableDatabase.delete("tasks", "id = ?", arrayOf(id))
    }

    fun markTaskDeleted(id: String) {
        writableDatabase.update("tasks", ContentValues().apply {
            put("deleted", 1)
            put("dirty", 1)
        }, "id = ?", arrayOf(id))
    }

    fun dirtyTasks(): List<LocalTask> {
        val db = readableDatabase
        val out = mutableListOf<LocalTask>()
        db.query("tasks", null, "dirty = 1", null, null, null, null).use { c ->
            while (c.moveToNext()) out.add(readTask(c))
        }
        return out
    }

    /**
     * Every task id the DB knows, including soft-deleted tombstones. Used to
     * cancel pending planner alarms for tasks that no longer exist.
     */
    fun queryAllTaskIds(): List<String> {
        val db = readableDatabase
        val out = mutableListOf<String>()
        db.query("tasks", arrayOf("id"), null, null, null, null, null).use { c ->
            while (c.moveToNext()) out.add(c.getString(0))
        }
        return out
    }

    private fun readTask(c: Cursor): LocalTask = LocalTask(
        id = c.getString(c.getColumnIndexOrThrow("id")),
        title = c.getString(c.getColumnIndexOrThrow("title")),
        status = c.getString(c.getColumnIndexOrThrow("status")),
        recurrence_days = decodeDays(c.getString(c.getColumnIndexOrThrow("recurrence_days"))),
        list_id = c.getString(c.getColumnIndexOrThrow("list_id")),
        start_time = c.getString(c.getColumnIndexOrThrow("start_time")),
        end_time = c.getString(c.getColumnIndexOrThrow("end_time")),
        color = c.getString(c.getColumnIndexOrThrow("color")),
        scheduled_date = c.getString(c.getColumnIndexOrThrow("scheduled_date")),
        alarm_minutes = c.getInt(c.getColumnIndexOrThrow("alarm_minutes")).takeIf { !c.isNull(c.getColumnIndexOrThrow("alarm_minutes")) },
        created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
        updated_at = c.getString(c.getColumnIndexOrThrow("updated_at")),
        dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
        deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1,
        synced = c.getInt(c.getColumnIndexOrThrow("synced")) == 1
    )



    // --- completions ---

    fun upsertCompletions(completions: List<LocalCompletion>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (c in completions) {
                db.insertWithOnConflict(
                    "completions", null,
                    ContentValues().apply {
                        put("id", c.id)
                        put("task_id", c.task_id)
                        put("completed_date", c.completed_date)
                        put("created_at", c.created_at)
                        put("dirty", if (c.dirty) 1 else 0)
                        put("deleted", if (c.deleted) 1 else 0)
                    },
                    SQLiteDatabase.CONFLICT_REPLACE
                )
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun queryCompletions(): List<LocalCompletion> {
        val db = readableDatabase
        val out = mutableListOf<LocalCompletion>()
        db.query("completions", null, "deleted = 0", null, null, null, "completed_date ASC").use { c ->
            while (c.moveToNext()) {
                out += LocalCompletion(
                    id = c.getString(c.getColumnIndexOrThrow("id")),
                    task_id = c.getString(c.getColumnIndexOrThrow("task_id")),
                    completed_date = c.getString(c.getColumnIndexOrThrow("completed_date")),
                    created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
                    dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1
                )
            }
        }
        return out
    }

    /** Returns a completion row regardless of its `deleted` flag (null when absent). */
    fun queryCompletionIncludingDeleted(id: String): LocalCompletion? {
        val db = readableDatabase
        db.query("completions", null, "id = ?", arrayOf(id), null, null, null).use { c ->
            return if (c.moveToFirst()) {
                LocalCompletion(
                    id = c.getString(c.getColumnIndexOrThrow("id")),
                    task_id = c.getString(c.getColumnIndexOrThrow("task_id")),
                    completed_date = c.getString(c.getColumnIndexOrThrow("completed_date")),
                    created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
                    dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1
                )
            } else null
        }
    }

    /** All locally-changed completions (including tombstones) awaiting push. */
    fun dirtyCompletions(): List<LocalCompletion> {
        val db = readableDatabase
        val out = mutableListOf<LocalCompletion>()
        db.query("completions", null, "dirty = 1", null, null, null, null).use { c ->
            while (c.moveToNext()) {
                out += LocalCompletion(
                    id = c.getString(c.getColumnIndexOrThrow("id")),
                    task_id = c.getString(c.getColumnIndexOrThrow("task_id")),
                    completed_date = c.getString(c.getColumnIndexOrThrow("completed_date")),
                    created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
                    dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1
                )
            }
        }
        return out
    }

    fun markCompletionDirty(id: String) {
        writableDatabase.update("completions", ContentValues().apply { put("dirty", 1) }, "id = ?", arrayOf(id))
    }

    fun markCompletionClean(id: String) {
        writableDatabase.update("completions", ContentValues().apply { put("dirty", 0) }, "id = ?", arrayOf(id))
    }

    fun markCompletionDeleted(id: String) {
        writableDatabase.update("completions", ContentValues().apply {
            put("deleted", 1)
            put("dirty", 1)
        }, "id = ?", arrayOf(id))
    }


    // --- task lists ---

    fun upsertTaskLists(lists: List<LocalTaskList>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (l in lists) {
                db.insertWithOnConflict(
                    "task_lists", null,
                    ContentValues().apply {
                        put("id", l.id)
                        put("name", l.name)
                        put("description", l.description)
                        put("owner_id", l.owner_id)
                        put("created_at", l.created_at)
                        put("updated_at", l.updated_at)
                        put("dirty", if (l.dirty) 1 else 0)
                        put("deleted", if (l.deleted) 1 else 0)
                        put("synced", if (l.synced) 1 else 0)
                    },
                    SQLiteDatabase.CONFLICT_REPLACE
                )
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun queryTaskLists(): List<LocalTaskList> {
        val db = readableDatabase
        val out = mutableListOf<LocalTaskList>()
        db.query("task_lists", null, "deleted = 0", null, null, null, "created_at DESC").use { c ->
            while (c.moveToNext()) {
                out += LocalTaskList(
                    id = c.getString(c.getColumnIndexOrThrow("id")),
                    name = c.getString(c.getColumnIndexOrThrow("name")),
                    description = c.getString(c.getColumnIndexOrThrow("description")),
                    owner_id = c.getString(c.getColumnIndexOrThrow("owner_id")),
                    created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
                    updated_at = c.getString(c.getColumnIndexOrThrow("updated_at")),
                    dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1,
                    synced = c.getInt(c.getColumnIndexOrThrow("synced")) == 1
                )
            }
        }
        return out
    }

    /** Returns a task-list row regardless of its `deleted` flag (null when absent). */
    fun queryTaskListIncludingDeleted(id: String): LocalTaskList? {
        val db = readableDatabase
        db.query("task_lists", null, "id = ?", arrayOf(id), null, null, null).use { c ->
            return if (c.moveToFirst()) {
                LocalTaskList(
                    id = c.getString(c.getColumnIndexOrThrow("id")),
                    name = c.getString(c.getColumnIndexOrThrow("name")),
                    description = c.getString(c.getColumnIndexOrThrow("description")),
                    owner_id = c.getString(c.getColumnIndexOrThrow("owner_id")),
                    created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
                    updated_at = c.getString(c.getColumnIndexOrThrow("updated_at")),
                    dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1,
                    synced = c.getInt(c.getColumnIndexOrThrow("synced")) == 1
                )
            } else null
        }
    }

    fun markTaskListDirty(id: String, dirty: Boolean = true) {
        writableDatabase.update("task_lists", ContentValues().apply { put("dirty", if (dirty) 1 else 0) }, "id = ?", arrayOf(id))
    }

    /** Marks a task list as confirmed on the server (a later edit should UPDATE). */
    fun markTaskListSynced(id: String) {
        writableDatabase.update("task_lists", ContentValues().apply { put("synced", 1) }, "id = ?", arrayOf(id))
    }

    /** Hard-deletes a local row (used for never-synced tombstones). */
    fun deleteLocalTaskList(id: String) {
        writableDatabase.delete("task_lists", "id = ?", arrayOf(id))
    }

    fun markTaskListDeleted(id: String) {
        writableDatabase.update("task_lists", ContentValues().apply {
            put("deleted", 1)
            put("dirty", 1)
        }, "id = ?", arrayOf(id))
    }

    // --- shares ---

    fun upsertShares(shares: List<LocalShare>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (s in shares) {
                db.insertWithOnConflict(
                    "shares", null,
                    ContentValues().apply {
                        put("list_id", s.list_id)
                        put("shared_with", s.shared_with)
                        put("role", s.role)
                        put("created_at", s.created_at)
                        put("deleted", if (s.deleted) 1 else 0)
                    },
                    SQLiteDatabase.CONFLICT_REPLACE
                )
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun queryShares(): List<LocalShare> {
        val db = readableDatabase
        val out = mutableListOf<LocalShare>()
        db.query("shares", null, "deleted = 0", null, null, null, null).use { c ->
            while (c.moveToNext()) {
                out += LocalShare(
                    list_id = c.getString(c.getColumnIndexOrThrow("list_id")),
                    shared_with = c.getString(c.getColumnIndexOrThrow("shared_with")),
                    role = c.getString(c.getColumnIndexOrThrow("role")),
                    created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1
                )
            }
        }
        return out
    }

    /** Applies a tombstone key "list_id:shared_with" to a local share row. */
    fun markShareDeleted(key: String) {
        val parts = key.split(":")
        if (parts.size != 2) return
        writableDatabase.update("shares", ContentValues().apply { put("deleted", 1) }, "list_id = ? AND shared_with = ?", parts.toTypedArray())
    }

    // --- tombstones (server deletions) ---

    fun applyDeletedTaskIds(ids: List<String>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (id in ids) {
                db.delete("tasks", "id = ? AND dirty = 0", arrayOf(id))
                db.update("tasks", ContentValues().apply { put("deleted", 1) }, "id = ? AND dirty = 1", arrayOf(id))
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun applyDeletedCompletionIds(ids: List<String>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (id in ids) {
                db.delete("completions", "id = ? AND dirty = 0", arrayOf(id))
                db.update("completions", ContentValues().apply { put("deleted", 1) }, "id = ? AND dirty = 1", arrayOf(id))
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }

    fun applyDeletedTaskListIds(ids: List<String>) {
        val db = writableDatabase
        db.beginTransaction()
        try {
            for (id in ids) {
                db.delete("task_lists", "id = ? AND dirty = 0", arrayOf(id))
                db.update("task_lists", ContentValues().apply { put("deleted", 1) }, "id = ? AND dirty = 1", arrayOf(id))
            }
            db.setTransactionSuccessful()
        } finally {
            db.endTransaction()
        }
    }
}

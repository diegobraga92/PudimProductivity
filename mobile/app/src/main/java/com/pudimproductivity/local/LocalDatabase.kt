package com.pudimproductivity.local

import android.content.ContentValues
import android.content.Context
import android.database.Cursor
import android.database.sqlite.SQLiteDatabase
import android.database.sqlite.SQLiteOpenHelper

/**
 * Local SQLite database for offline-first operation (Phase 9c).
 *
 * Chosen over Room because AGP 9.1's built-in Kotlin is incompatible with KSP
 * (Room's annotation processor) and migrating the whole build off built-in
 * Kotlin is unsupported by Kotlin 2.2.21 (see docs/adr/012). This exposes a
 * Room-style DAO API — `upsertTasks`, `queryTasks`, `markDirty` — with zero
 * annotation processing, so the offline architecture is identical to a Room
 * design and can be swapped to Room when the toolchain allows.
 *
 * Every table carries `dirty` (local change awaiting push) and `deleted`
 * (local or server tombstone) so the SyncManager can converge with the backend.
 */
class LocalDatabase(context: Context) : SQLiteOpenHelper(context.applicationContext, DB_NAME, null, DB_VERSION) {

    companion object {
        const val DB_NAME = "pudim_offline.db"
        const val DB_VERSION = 1

        const val META_KEY_LAST_SYNC = "last_sync_ts"
    }

    override fun onCreate(db: SQLiteDatabase) {
        db.execSQL("CREATE TABLE IF NOT EXISTS tasks (id TEXT PRIMARY KEY, title TEXT NOT NULL, status TEXT NOT NULL, recurrence_days TEXT, list_id TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL, dirty INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0)")
        db.execSQL("CREATE TABLE IF NOT EXISTS completions (id TEXT PRIMARY KEY, task_id TEXT NOT NULL, completed_date TEXT NOT NULL, created_at TEXT NOT NULL, dirty INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0)")
        db.execSQL("CREATE TABLE IF NOT EXISTS task_lists (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', owner_id TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, dirty INTEGER NOT NULL DEFAULT 0, deleted INTEGER NOT NULL DEFAULT 0)")
        db.execSQL("CREATE TABLE IF NOT EXISTS shares (list_id TEXT NOT NULL, shared_with TEXT NOT NULL, role TEXT NOT NULL, created_at TEXT NOT NULL, deleted INTEGER NOT NULL DEFAULT 0, PRIMARY KEY (list_id, shared_with))")
        db.execSQL("CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)")
        db.execSQL("CREATE INDEX IF NOT EXISTS idx_tasks_deleted ON tasks (deleted)")
        db.execSQL("CREATE INDEX IF NOT EXISTS idx_completions_deleted ON completions (deleted)")
        db.execSQL("CREATE INDEX IF NOT EXISTS idx_task_lists_deleted ON task_lists (deleted)")
    }

    override fun onUpgrade(db: SQLiteDatabase, oldVersion: Int, newVersion: Int) {
        // No migrations yet (version 1). Room-style fallback for future bumps.
        if (oldVersion < 2) {
            // Reserved for future schema changes.
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
                    put("created_at", t.created_at)
                    put("updated_at", t.updated_at)
                    put("dirty", if (t.dirty) 1 else 0)
                    put("deleted", if (t.deleted) 1 else 0)
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

    fun markTaskDirty(id: String, dirty: Boolean = true) {
        writableDatabase.update("tasks", ContentValues().apply { put("dirty", if (dirty) 1 else 0) }, "id = ?", arrayOf(id))
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

    private fun readTask(c: Cursor): LocalTask = LocalTask(
        id = c.getString(c.getColumnIndexOrThrow("id")),
        title = c.getString(c.getColumnIndexOrThrow("title")),
        status = c.getString(c.getColumnIndexOrThrow("status")),
        recurrence_days = decodeDays(c.getString(c.getColumnIndexOrThrow("recurrence_days"))),
        list_id = c.getString(c.getColumnIndexOrThrow("list_id")),
        created_at = c.getString(c.getColumnIndexOrThrow("created_at")),
        updated_at = c.getString(c.getColumnIndexOrThrow("updated_at")),
        dirty = c.getInt(c.getColumnIndexOrThrow("dirty")) == 1,
        deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1
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
                    deleted = c.getInt(c.getColumnIndexOrThrow("deleted")) == 1
                )
            }
        }
        return out
    }

    fun markTaskListDirty(id: String, dirty: Boolean = true) {
        writableDatabase.update("task_lists", ContentValues().apply { put("dirty", if (dirty) 1 else 0) }, "id = ?", arrayOf(id))
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

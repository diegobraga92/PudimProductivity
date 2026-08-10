package com.pudimproductivity.local

/**
 * Local entity mirroring the API `Task` shape, plus offline-sync bookkeeping:
 * `dirty` (1 = local change not yet pushed) and `deleted` (1 = local tombstone).
 */
data class LocalTask(
    val id: String,
    val title: String,
    val status: String,
    val recurrence_days: List<String>? = null,
    val list_id: String? = null,
    val created_at: String,
    val updated_at: String,
    val dirty: Boolean = false,
    val deleted: Boolean = false
)

data class LocalCompletion(
    val id: String,
    val task_id: String,
    val completed_date: String,
    val created_at: String,
    val dirty: Boolean = false,
    val deleted: Boolean = false
)

data class LocalTaskList(
    val id: String,
    val name: String,
    val description: String = "",
    val owner_id: String = "",
    val created_at: String,
    val updated_at: String,
    val dirty: Boolean = false,
    val deleted: Boolean = false
)

data class LocalShare(
    val list_id: String,
    val shared_with: String,
    val role: String,
    val created_at: String,
    val deleted: Boolean = false
)

/** Inline JSON (de)serialization for recurrence_days. */
fun encodeDays(days: List<String>?): String = (days ?: emptyList()).joinToString(",", prefix = "[", postfix = "]")

fun decodeDays(raw: String?): List<String>? {
    if (raw.isNullOrBlank() || raw == "[]") return null
    return raw
        .removeSurrounding("[", "]")
        .split(",")
        .map { it.trim() }
        .filter { it.isNotEmpty() }
}

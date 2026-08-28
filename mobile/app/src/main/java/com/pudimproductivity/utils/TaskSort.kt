package com.pudimproductivity.utils

import com.pudimproductivity.api.Task

/**
 * Task ordering options.
 */
enum class SortOption(val key: String, val label: String) {
    ALPHA_ASC("alpha-asc", "Name A-Z"),
    ALPHA_DESC("alpha-desc", "Name Z-A"),
    CREATED_ASC("created-asc", "Oldest first"),
    CREATED_DESC("created-desc", "Newest first"),
    TIME_ASC("time-asc", "Time ↑"),
    TIME_DESC("time-desc", "Time ↓");

    companion object {
        val default: SortOption = CREATED_DESC

        fun fromKey(key: String?): SortOption = entries.firstOrNull { it.key == key } ?: default
    }
}

/**
 * Orders [tasks] by [option]:
 *  - alpha: by title (case-insensitive)
 *  - created: by `created_at` (ISO-8601 string compare)
 *  - time: scheduled tasks (with `start_time`) first, sorted by time;
 *    unscheduled tasks are appended in their original order.
 */
fun sortTasks(tasks: List<Task>, option: SortOption): List<Task> = when (option) {
    SortOption.ALPHA_ASC -> tasks.sortedBy { it.title.lowercase() }
    SortOption.ALPHA_DESC -> tasks.sortedByDescending { it.title.lowercase() }
    SortOption.CREATED_ASC -> tasks.sortedBy { it.created_at }
    SortOption.CREATED_DESC -> tasks.sortedByDescending { it.created_at }
    SortOption.TIME_ASC -> sortByStartTime(tasks, ascending = true)
    SortOption.TIME_DESC -> sortByStartTime(tasks, ascending = false)
}

private fun sortByStartTime(tasks: List<Task>, ascending: Boolean): List<Task> {
    val scheduled = tasks.filter { it.start_time != null }
    val unscheduled = tasks.filter { it.start_time == null }
    val sorted = if (ascending) {
        scheduled.sortedBy { it.start_time }
    } else {
        scheduled.sortedByDescending { it.start_time }
    }
    return sorted + unscheduled
}

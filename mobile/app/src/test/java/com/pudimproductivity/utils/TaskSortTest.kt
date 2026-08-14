package com.pudimproductivity.utils

import com.pudimproductivity.api.Task
import org.junit.Assert.assertEquals
import org.junit.Test

/** JVM unit tests for the task ordering logic (mirrors web/src/utils/sort.ts). */
class TaskSortTest {

    private fun task(
        id: String,
        title: String,
        createdAt: String,
        startTime: String? = null
    ) = Task(
        id = id,
        title = title,
        status = "todo",
        created_at = createdAt,
        updated_at = createdAt,
        start_time = startTime
    )

    private fun titles(tasks: List<Task>): List<String> = tasks.map { it.title }

    @Test
    fun `created-desc orders newest first`() {
        val tasks = listOf(
            task("1", "Old", "2026-01-01T08:00:00Z"),
            task("2", "New", "2026-01-03T08:00:00Z"),
            task("3", "Middle", "2026-01-02T08:00:00Z")
        )
        assertEquals(
            listOf("New", "Middle", "Old"),
            titles(sortTasks(tasks, SortOption.CREATED_DESC))
        )
    }

    @Test
    fun `created-asc orders oldest first`() {
        val tasks = listOf(
            task("1", "Old", "2026-01-01T08:00:00Z"),
            task("2", "New", "2026-01-03T08:00:00Z"),
            task("3", "Middle", "2026-01-02T08:00:00Z")
        )
        assertEquals(
            listOf("Old", "Middle", "New"),
            titles(sortTasks(tasks, SortOption.CREATED_ASC))
        )
    }

    @Test
    fun `alpha-asc orders by title a to z case-insensitively`() {
        val tasks = listOf(
            task("1", "Zebra", "2026-01-01T08:00:00Z"),
            task("2", "apple", "2026-01-01T08:00:00Z"),
            task("3", "Banana", "2026-01-01T08:00:00Z")
        )
        assertEquals(
            listOf("apple", "Banana", "Zebra"),
            titles(sortTasks(tasks, SortOption.ALPHA_ASC))
        )
    }

    @Test
    fun `alpha-desc orders by title z to a`() {
        val tasks = listOf(
            task("1", "Zebra", "2026-01-01T08:00:00Z"),
            task("2", "apple", "2026-01-01T08:00:00Z"),
            task("3", "Banana", "2026-01-01T08:00:00Z")
        )
        assertEquals(
            listOf("Zebra", "Banana", "apple"),
            titles(sortTasks(tasks, SortOption.ALPHA_DESC))
        )
    }

    @Test
    fun `time-asc puts scheduled before unscheduled and sorts by start time`() {
        val tasks = listOf(
            task("1", "No time", "2026-01-01T08:00:00Z"),
            task("2", "Late", "2026-01-01T08:00:00Z", startTime = "10:00"),
            task("3", "Early", "2026-01-01T08:00:00Z", startTime = "09:00")
        )
        assertEquals(
            listOf("Early", "Late", "No time"),
            titles(sortTasks(tasks, SortOption.TIME_ASC))
        )
    }

    @Test
    fun `time-desc puts scheduled before unscheduled and sorts by start time descending`() {
        val tasks = listOf(
            task("1", "No time", "2026-01-01T08:00:00Z"),
            task("2", "Late", "2026-01-01T08:00:00Z", startTime = "10:00"),
            task("3", "Early", "2026-01-01T08:00:00Z", startTime = "09:00")
        )
        assertEquals(
            listOf("Late", "Early", "No time"),
            titles(sortTasks(tasks, SortOption.TIME_DESC))
        )
    }

    @Test
    fun `time sort keeps unscheduled tasks in their original order`() {
        val tasks = listOf(
            task("1", "First unscheduled", "2026-01-01T08:00:00Z"),
            task("2", "Scheduled", "2026-01-01T08:00:00Z", startTime = "08:00"),
            task("3", "Second unscheduled", "2026-01-01T08:00:00Z")
        )
        assertEquals(
            listOf("Scheduled", "First unscheduled", "Second unscheduled"),
            titles(sortTasks(tasks, SortOption.TIME_ASC))
        )
    }

    @Test
    fun `fromKey maps persisted keys and falls back to default`() {
        assertEquals(SortOption.CREATED_ASC, SortOption.fromKey("created-asc"))
        assertEquals(SortOption.TIME_DESC, SortOption.fromKey("time-desc"))
        assertEquals(SortOption.default, SortOption.fromKey("bogus"))
        assertEquals(SortOption.default, SortOption.fromKey(null))
    }
}

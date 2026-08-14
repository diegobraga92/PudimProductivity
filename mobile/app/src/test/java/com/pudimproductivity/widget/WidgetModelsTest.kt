package com.pudimproductivity.widget

import com.pudimproductivity.local.LocalCompletion
import com.pudimproductivity.local.LocalTask
import java.time.LocalDate
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * JVM unit tests for the pure widget mapping logic (Phase 10). No Android
 * dependencies — the `build*` functions take plain local-DB entities.
 */
class WidgetModelsTest {

    private fun task(
        id: String,
        title: String,
        status: String = "todo",
        recurrenceDays: List<String>? = null,
        deleted: Boolean = false
    ) = LocalTask(
        id = id,
        title = title,
        status = status,
        recurrence_days = recurrenceDays,
        created_at = "2026-01-01T00:00:00Z",
        updated_at = "2026-01-01T00:00:00Z",
        dirty = false,
        deleted = deleted
    )

    private fun completion(taskId: String, date: String, deleted: Boolean = false) =
        LocalCompletion(
            id = "$taskId-$date",
            task_id = taskId,
            completed_date = date,
            created_at = "2026-01-01T00:00:00Z",
            dirty = false,
            deleted = deleted
        )

    @Test
    fun `tasks snapshot excludes habits and deleted rows`() {
        val snapshot = buildTasksSnapshot(
            listOf(
                task("1", "Buy milk"),
                task("2", "Workout", recurrenceDays = listOf("mon")),
                task("3", "Gone", deleted = true),
                task("4", "Read", status = "done")
            )
        )

        assertEquals(2, snapshot.total)
        assertEquals(1, snapshot.done)
        assertEquals(1, snapshot.pending.size)
        assertEquals(1, snapshot.remaining)
        assertEquals("Buy milk", snapshot.pending.first().title)
        assertFalse(snapshot.pending.any { it.id == "4" })
    }

    @Test
    fun `tasks snapshot sorts pending first then by title`() {
        val snapshot = buildTasksSnapshot(
            listOf(
                task("1", "Zebra", status = "done"),
                task("2", "Apple"),
                task("3", "Banana")
            )
        )

        assertEquals(listOf("Apple", "Banana"), snapshot.pending.map { it.title })
        assertEquals(1, snapshot.done)
        assertEquals(2, snapshot.remaining)
    }

    @Test
    fun `tasks snapshot remaining is zero when everything is done`() {
        val snapshot = buildTasksSnapshot(
            listOf(
                task("1", "Read", status = "done"),
                task("2", "Run", status = "done")
            )
        )

        assertEquals(0, snapshot.remaining)
        assertEquals(2, snapshot.done)
        assertTrue(snapshot.pending.isEmpty())
    }

    @Test
    fun `habits snapshot shows scheduled and completed-today habits`() {
        val today = LocalDate.now()
        val todayDay = dayName(today)

        val snapshot = buildHabitsSnapshot(
            tasks = listOf(
                task("h1", "Workout", recurrenceDays = listOf(todayDay)),
                task("h2", "Read", recurrenceDays = listOf("sat")),          // not today, not done
                task("h3", "Meditate", recurrenceDays = listOf("sat")),      // not today but done
                task("h4", "Deleted habit", recurrenceDays = listOf(todayDay), deleted = true)
            ),
            completions = listOf(completion("h3", today.toString())),
            today = today.toString(),
            todayDay = todayDay
        )

        assertEquals(setOf("h1", "h3"), snapshot.habits.map { it.id }.toSet())
        assertEquals(2, snapshot.scheduledToday)
        assertEquals(1, snapshot.doneToday)
        assertFalse(snapshot.habits.first { it.id == "h1" }.completedToday)
        assertTrue(snapshot.habits.first { it.id == "h3" }.completedToday)
    }

    @Test
    fun `habits snapshot ignores deleted completions`() {
        val today = LocalDate.now()
        val todayDay = dayName(today)

        val snapshot = buildHabitsSnapshot(
            tasks = listOf(task("h1", "Run", recurrenceDays = listOf(todayDay))),
            completions = listOf(
                completion("h1", today.toString()),
                completion("h1", today.toString(), deleted = true)
            ),
            today = today.toString(),
            todayDay = todayDay
        )

        assertEquals(1, snapshot.doneToday)
        assertTrue(snapshot.habits.first().completedToday)
    }

    @Test
    fun `habits snapshot computes current streak`() {
        val today = LocalDate.now()

        val snapshot = buildHabitsSnapshot(
            tasks = listOf(task("h1", "Run", recurrenceDays = listOf(dayName(today)))),
            completions = listOf(
                completion("h1", today.toString()),
                completion("h1", today.minusDays(1).toString()),
                completion("h1", today.minusDays(2).toString())
            ),
            today = today.toString(),
            todayDay = dayName(today)
        )

        assertEquals(3, snapshot.habits.first().streak)
        assertEquals(3, snapshot.habits.first().bestStreak)
    }

    @Test
    fun `habits snapshot keeps the best streak even when longer than the current one`() {
        val today = LocalDate.now()

        val snapshot = buildHabitsSnapshot(
            tasks = listOf(task("h1", "Run", recurrenceDays = listOf(dayName(today)))),
            completions = listOf(
                // Current run: today + yesterday.
                completion("h1", today.toString()),
                completion("h1", today.minusDays(1).toString()),
                // Longer run that ended a few days ago — no longer contiguous.
                completion("h1", today.minusDays(6).toString()),
                completion("h1", today.minusDays(7).toString()),
                completion("h1", today.minusDays(8).toString()),
                completion("h1", today.minusDays(9).toString()),
                completion("h1", today.minusDays(10).toString())
            ),
            today = today.toString(),
            todayDay = dayName(today)
        )

        val row = snapshot.habits.first()
        assertEquals(2, row.streak)
        assertEquals(5, row.bestStreak)
    }

    @Test
    fun `habits snapshot hides streak badge when there is no streak`() {
        val today = LocalDate.now()

        val snapshot = buildHabitsSnapshot(
            tasks = listOf(task("h1", "Run", recurrenceDays = listOf(dayName(today)))),
            completions = listOf(
                completion("h1", today.minusDays(3).toString())
            ),
            today = today.toString(),
            todayDay = dayName(today)
        )

        val row = snapshot.habits.first()
        assertEquals(0, row.streak)
        assertEquals(1, row.bestStreak)
    }

    @Test
    fun `day name is monday-first`() {
        // 2026-08-10 is a Monday.
        assertEquals("mon", dayName(LocalDate.of(2026, 8, 10)))
        assertEquals("tue", dayName(LocalDate.of(2026, 8, 11)))
        assertEquals("sun", dayName(LocalDate.of(2026, 8, 16)))
    }
}

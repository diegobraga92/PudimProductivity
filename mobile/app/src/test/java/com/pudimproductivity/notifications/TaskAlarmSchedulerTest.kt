package com.pudimproductivity.notifications

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test
import java.time.ZoneOffset
import java.time.ZonedDateTime

class TaskAlarmSchedulerTest {

    // Fixed instants: 2026-01-05 is a Monday.
    private val monday845 = ZonedDateTime.of(2026, 1, 5, 8, 45, 0, 0, ZoneOffset.UTC)
    private val monday855 = ZonedDateTime.of(2026, 1, 5, 8, 55, 0, 0, ZoneOffset.UTC)
    private val saturday1000 = ZonedDateTime.of(2026, 1, 10, 10, 0, 0, 0, ZoneOffset.UTC)

    @Test
    fun `next alarm lands on a recurrence day at start minus alarm minutes`() {
        val next = TaskAlarmScheduler.nextAlarmDateTime(
            startTime = "09:00",
            alarmMinutes = 10,
            recurrenceDays = listOf("mon", "wed"),
            now = monday845
        )
        // Monday 08:50 is still ahead of 08:45 → same day.
        assertEquals(ZonedDateTime.of(2026, 1, 5, 8, 50, 0, 0, ZoneOffset.UTC), next)
    }

    @Test
    fun `next alarm rolls to the following week when today's occurrence already passed`() {
        val next = TaskAlarmScheduler.nextAlarmDateTime(
            startTime = "09:00",
            alarmMinutes = 10,
            recurrenceDays = listOf("mon"),
            now = monday855
        )
        // Monday 08:50 is in the past → next Monday.
        assertEquals(ZonedDateTime.of(2026, 1, 12, 8, 50, 0, 0, ZoneOffset.UTC), next)
    }

    @Test
    fun `next alarm picks the nearest future recurrence day`() {
        val next = TaskAlarmScheduler.nextAlarmDateTime(
            startTime = "08:00",
            alarmMinutes = 15,
            recurrenceDays = listOf("sun"),
            now = saturday1000
        )
        assertEquals(ZonedDateTime.of(2026, 1, 11, 7, 45, 0, 0, ZoneOffset.UTC), next)
    }

    @Test
    fun `alarm offset crossing midnight returns null`() {
        assertNull(
            TaskAlarmScheduler.nextAlarmDateTime(
                startTime = "00:10",
                alarmMinutes = 20,
                recurrenceDays = listOf("mon"),
                now = monday845
            )
        )
    }

    @Test
    fun `parseTimeToMinutes handles valid and malformed input`() {
        assertEquals(570, TaskAlarmScheduler.parseTimeToMinutes("09:30"))
        assertEquals(9 * 60, TaskAlarmScheduler.parseTimeToMinutes("09"))
        assertNull(TaskAlarmScheduler.parseTimeToMinutes("ab:cd"))
        assertNull(TaskAlarmScheduler.parseTimeToMinutes("25:00"))
    }
}

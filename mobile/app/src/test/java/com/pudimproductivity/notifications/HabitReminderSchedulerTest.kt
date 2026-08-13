package com.pudimproductivity.notifications

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.ZoneId
import java.time.ZonedDateTime
import java.time.temporal.ChronoUnit

class HabitReminderSchedulerTest {

    @Test
    fun `millisUntilNext lands on the next 8am local`() {
        val now = ZonedDateTime.of(2026, 8, 13, 9, 30, 0, 0, ZoneId.systemDefault())
        val millis = HabitReminderScheduler.millisUntilNext(8, now)

        val target = now.plus(millis, ChronoUnit.MILLIS)
        assertTrue(millis > 0)
        assertTrue(target.isAfter(now))
        assertEquals(8, target.hour)
        assertEquals(0, target.minute)
    }

    @Test
    fun `millisUntilNext rolls to the next day when now is exactly on the hour`() {
        val now = ZonedDateTime.of(2026, 8, 13, 8, 0, 0, 0, ZoneId.systemDefault())
        val millis = HabitReminderScheduler.millisUntilNext(8, now)

        val target = now.plus(millis, ChronoUnit.MILLIS)
        assertTrue(millis > 0)
        assertEquals(2026, target.year)
        assertEquals(8, target.monthValue)
        assertEquals(14, target.dayOfMonth)
        assertEquals(8, target.hour)
        assertEquals(0, target.minute)
    }

    @Test
    fun `millisUntilNext always returns a positive delay`() {
        val now = ZonedDateTime.of(2026, 8, 13, 12, 0, 0, 0, ZoneId.systemDefault())
        assertTrue(HabitReminderScheduler.millisUntilNext(7, now) > 0)
        assertTrue(HabitReminderScheduler.millisUntilNext(23, now) > 0)
    }
}

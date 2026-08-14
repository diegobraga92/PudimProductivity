package com.pudimproductivity.utils

import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.temporal.TemporalAdjusters

/**
 * Returns 7 date strings (YYYY-MM-DD) for the week starting Monday,
 * offset by [weekOffset] weeks (0 = current week, -1 = last week, etc.).
 */
fun getWeekDates(weekOffset: Int = 0): List<String> {
    val today = LocalDate.now()
    val monday = today.with(TemporalAdjusters.previousOrSame(DayOfWeek.MONDAY))
        .plusWeeks(weekOffset.toLong())
    return (0..6).map { monday.plusDays(it.toLong()).format(DateTimeFormatter.ISO_LOCAL_DATE) }
}

/**
 * Returns today's date as YYYY-MM-DD.
 */
fun getToday(): String {
    return LocalDate.now().format(DateTimeFormatter.ISO_LOCAL_DATE)
}

/**
 * Returns 7 ISO date strings (YYYY-MM-DD) for a rolling 7-day window anchored
 * to today (mirrors the web's `getRollingWindowDates`).
 *
 * Unlike [getWeekDates] (fixed Monday–Sunday), offset 0 returns the last 7 days
 * ending today, with today as the final column. This keeps habit streaks flowing
 * continuously between calendar weeks — on Monday you still see the previous
 * week's completions instead of a hard reset to a blank Monday–Sunday grid.
 *
 * @param offset Offset relative to the current window:
 *   0 (default) = last 7 days ending today,
 *  -1 = the 7 days before that,
 *  +1 = the 7 days after that (future — usually not shown).
 */
fun getRollingWindowDates(offset: Int = 0): List<String> {
    val end = LocalDate.now().plusWeeks(offset.toLong())
    val start = end.minusDays(6)
    return (0..6).map { start.plusDays(it.toLong()).format(DateTimeFormatter.ISO_LOCAL_DATE) }
}

/**
 * Returns the date `dayOffset` days from today as YYYY-MM-DD (used by the
 * single-day Planner, where 0 = today, -1 = yesterday, +1 = tomorrow).
 */
fun getDate(dayOffset: Int = 0): String {
    return LocalDate.now().plusDays(dayOffset.toLong()).format(DateTimeFormatter.ISO_LOCAL_DATE)
}

/**
 * Given a list of 7 ISO date strings (Mon–Sun), returns a human-friendly
 * range like "Mar 15–21". Month abbreviations can be localized via [monthNames]
 * (defaults to English).
 */
fun formatWeekRange(
    dates: List<String>,
    monthNames: List<String> = listOf("Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec")
): String {
    if (dates.size < 2) return ""
    val start = formatShortDate(dates.first(), monthNames)
    val end = formatShortDate(dates.last(), monthNames)
    // Same month → "Mar 15–21", different months → "Mar 28–Apr 3"
    val startMonth = dates.first().substring(5, 7)
    val endMonth = dates.last().substring(5, 7)
    return if (startMonth == endMonth) {
        "${monthNames[startMonth.toInt() - 1]} ${start.substringAfter(" ")}–${end.substringAfter(" ")}"
    } else {
        "$start – $end"
    }
}

private fun formatShortDate(isoDate: String, monthNames: List<String>): String {
    val date = LocalDate.parse(isoDate)
    return "${monthNames[date.monthValue - 1]} ${date.dayOfMonth}"
}
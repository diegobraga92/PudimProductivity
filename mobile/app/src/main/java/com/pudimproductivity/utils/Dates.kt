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
 * Given a list of 7 ISO date strings (Mon–Sun), returns a human-friendly
 * range like "Mar 15–21".
 */
fun formatWeekRange(dates: List<String>): String {
    if (dates.size < 2) return ""
    val start = formatShortDate(dates.first())
    val end = formatShortDate(dates.last())
    // Same month → "Mar 15–21", different months → "Mar 28–Apr 3"
    val startMonth = dates.first().substring(5, 7)
    val endMonth = dates.last().substring(5, 7)
    return if (startMonth == endMonth) {
        "${formatMonth(dates.first())} ${start.substringAfter(" ")}–${end.substringAfter(" ")}"
    } else {
        "$start – $end"
    }
}

private fun formatShortDate(isoDate: String): String {
    val date = LocalDate.parse(isoDate)
    return date.format(DateTimeFormatter.ofPattern("MMM d"))
}

private fun formatMonth(isoDate: String): String {
    val date = LocalDate.parse(isoDate)
    return date.format(DateTimeFormatter.ofPattern("MMM"))
}
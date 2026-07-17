package com.pudimproductivity.utils

import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit

/**
 * Compute current and longest streak from a list of completion date strings (YYYY-MM-DD).
 * A streak counts consecutive days (including today if completed).
 */
data class StreakResult(val current: Int, val longest: Int)

fun computeStreaks(completions: List<String>): StreakResult {
    if (completions.isEmpty()) return StreakResult(0, 0)

    // Sort dates ascending and deduplicate
    val sorted = completions.sorted().distinct()

    // Build a set for O(1) lookup
    val dateSet = sorted.toSet()

    // Find the longest streak by scanning consecutive days
    var longest = 0
    var tempStreak = 0

    for (i in sorted.indices) {
        if (i == 0) {
            tempStreak = 1
        } else {
            val prev = LocalDate.parse(sorted[i - 1])
            val curr = LocalDate.parse(sorted[i])
            val diffDays = ChronoUnit.DAYS.between(prev, curr).toInt()
            if (diffDays == 1) {
                tempStreak++
            } else {
                tempStreak = 1
            }
        }
        longest = maxOf(longest, tempStreak)
    }

    // Compute current streak: count backwards from today
    var current = 0
    val today = getToday()

    // If today is not completed, check if yesterday was (for "active" streak)
    var checkDate = LocalDate.now()
    if (!dateSet.contains(today)) {
        checkDate = checkDate.minusDays(1)
    }

    while (true) {
        val dateStr = checkDate.format(DateTimeFormatter.ISO_LOCAL_DATE)
        if (dateSet.contains(dateStr)) {
            current++
            checkDate = checkDate.minusDays(1)
        } else {
            break
        }
    }

    return StreakResult(current, longest)
}
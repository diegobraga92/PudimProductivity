package com.pudimproductivity.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pudimproductivity.utils.getToday
import com.pudimproductivity.utils.getWeekDates

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val DAY_SHORT = mapOf(
    "mon" to "M", "tue" to "T", "wed" to "W", "thu" to "T",
    "fri" to "F", "sat" to "S", "sun" to "S"
)

private fun getDayName(dateStr: String): String {
    val date = java.time.LocalDate.parse(dateStr)
    val dayIndex = (date.dayOfWeek.value + 6) % 7 // 0=Mon
    return DAY_ORDER[dayIndex]
}

/**
 * A 7-day habit completion heatmap with week navigation.
 *
 * @param recurrenceDays Days of the week this habit is scheduled (e.g. ["mon","wed","fri"])
 * @param completions List of ISO date strings (YYYY-MM-DA) for completed dates
 * @param onToggleDay Called with (date, isCompleted) when a day cell is tapped
 * @param disabled If true, disables all interactions
 * @param weekOffset Week offset (0 = current week, -1 = last week, etc.)
 * @param onWeekOffsetChange Called when user navigates to prev/next week
 */
@Composable
fun WeekHeatmap(
    recurrenceDays: List<String>,
    completions: List<String>,
    onToggleDay: (String, Boolean) -> Unit,
    disabled: Boolean = false,
    weekOffset: Int = 0,
    onWeekOffsetChange: ((Int) -> Unit)? = null
) {
    val weekDates = getWeekDates(weekOffset)
    val completedSet = completions.toSet()
    val today = getToday()
    Column {
        // Week navigation bar (only when the caller supplies a handler). Lists
        // of habit cards use a single shared WeekNavigator above the list
        // instead of one per card.
        if (onWeekOffsetChange != null) {
            WeekNavigator(
                weekOffset = weekOffset,
                onWeekOffsetChange = onWeekOffsetChange
            )
        }

        // Day cells row
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(4.dp)
        ) {
            weekDates.forEach { date ->
                val dayName = getDayName(date)
                val isScheduled = recurrenceDays.contains(dayName)
                val isCompleted = completedSet.contains(date)
                val isToday = date == today
                val canToggle = (isScheduled || isCompleted) && !disabled

                val cellColor = when {
                    isCompleted -> MaterialTheme.colorScheme.primary
                    isScheduled -> MaterialTheme.colorScheme.tertiaryContainer
                    else -> MaterialTheme.colorScheme.surfaceVariant
                }
                val textColor = when {
                    isCompleted -> MaterialTheme.colorScheme.onPrimary
                    isScheduled -> MaterialTheme.colorScheme.onTertiaryContainer
                    else -> MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
                }

                Box(
                    modifier = Modifier
                        .weight(1f)
                        .aspectRatio(1f)
                        .clip(CircleShape)
                        .background(cellColor)
                        .then(
                            if (canToggle) Modifier.clickable {
                                onToggleDay(date, isCompleted)
                            } else Modifier
                        ),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = DAY_SHORT[dayName] ?: "",
                        fontSize = 11.sp,
                        fontWeight = if (isToday) FontWeight.Bold else FontWeight.Normal,
                        color = textColor,
                        textAlign = TextAlign.Center
                    )
                }
            }
        }
    }
}
package com.pudimproductivity.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import com.pudimproductivity.utils.formatWeekRange
import com.pudimproductivity.utils.getWeekDates

/**
 * Single week selector shared by a list of habit cards (HabitScreen and the
 * Tasks screen's Habits tab). One navigator drives every heatmap in the list,
 * so the week only has to be chosen once instead of per card.
 */
@Composable
fun WeekNavigator(
    weekOffset: Int,
    onWeekOffsetChange: (Int) -> Unit,
    modifier: Modifier = Modifier
) {
    val weekDates = getWeekDates(weekOffset)
    val isCurrentWeek = weekOffset == 0

    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        TextButton(onClick = { onWeekOffsetChange(weekOffset - 1) }) {
            Text("← Prev", style = MaterialTheme.typography.labelSmall)
        }

        Text(
            text = if (isCurrentWeek) "This Week" else formatWeekRange(weekDates),
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        TextButton(
            onClick = { onWeekOffsetChange(weekOffset + 1) },
            enabled = weekOffset < 0
        ) {
            Text(
                "Next →",
                style = MaterialTheme.typography.labelSmall,
                color = if (weekOffset < 0) MaterialTheme.colorScheme.primary
                else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.38f)
            )
        }
    }
}

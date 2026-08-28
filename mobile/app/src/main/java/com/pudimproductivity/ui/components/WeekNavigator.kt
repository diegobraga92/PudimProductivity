package com.pudimproductivity.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.utils.formatWeekRange
import com.pudimproductivity.utils.getRollingWindowDates

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
    val weekDates = getRollingWindowDates(weekOffset)
    val isCurrentWindow = weekOffset == 0

    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        TextButton(onClick = { onWeekOffsetChange(weekOffset - 1) }) {
            Text("← " + Localization.text("week.prev"), style = MaterialTheme.typography.labelSmall)
        }

        Text(
            text = if (isCurrentWindow) Localization.text("week.thisWeek")
            else formatWeekRange(weekDates, Localization.months()),
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.SemiBold,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        TextButton(
            onClick = { onWeekOffsetChange(weekOffset + 1) },
            enabled = weekOffset < 0
        ) {
            Text(
                Localization.text("week.next") + " →",
                style = MaterialTheme.typography.labelSmall,
                color = if (weekOffset < 0) MaterialTheme.colorScheme.primary
                else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.38f)
            )
        }
    }
}

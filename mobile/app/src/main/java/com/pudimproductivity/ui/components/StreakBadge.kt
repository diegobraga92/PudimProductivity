package com.pudimproductivity.ui.components

import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.utils.StreakResult

/**
 * Displays a fire emoji with the current streak, and optionally the best streak.
 * If both current and longest are 0, nothing is rendered.
 */
@Composable
fun StreakBadge(
    current: Int,
    longest: Int,
    modifier: Modifier = Modifier
) {
    if (current == 0 && longest == 0) return

    Row(
        modifier = modifier,
        horizontalArrangement = Arrangement.spacedBy(2.dp)
    ) {
        Text(
            text = "🔥",
            fontSize = 14.sp
        )
        Text(
            text = current.toString(),
            style = MaterialTheme.typography.bodySmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onTertiaryContainer
        )
        if (longest > current) {
            Text(
                text = Localization.text("streak.best", "count" to longest),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.6f)
            )
        }
    }
}

/**
 * Convenience overload accepting a StreakResult.
 */
@Composable
fun StreakBadge(
    result: StreakResult,
    modifier: Modifier = Modifier
) = StreakBadge(current = result.current, longest = result.longest, modifier = modifier)
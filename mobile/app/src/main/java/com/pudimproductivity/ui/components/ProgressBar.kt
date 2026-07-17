package com.pudimproductivity.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * Variant colors for the progress bar.
 */
enum class ProgressVariant {
    TODO,
    HABIT,
    DEFAULT
}

/**
 * A horizontal progress bar with rounded corners.
 *
 * @param value Progress percentage (0–100)
 * @param variant Color variant
 * @param height Bar height in dp
 * @param modifier Modifier for the container
 */
@Composable
fun ProgressBar(
    value: Int,
    variant: ProgressVariant = ProgressVariant.DEFAULT,
    height: Dp = 8.dp,
    modifier: Modifier = Modifier
) {
    val clampedValue = value.coerceIn(0, 100)
    val fillColor = when (variant) {
        ProgressVariant.TODO -> Color(0xFF8B5CF6) // purple
        ProgressVariant.HABIT -> Color(0xFF10B981) // green
        ProgressVariant.DEFAULT -> MaterialTheme.colorScheme.primary
    }
    val trackColor = MaterialTheme.colorScheme.surfaceVariant

    Box(
        modifier = modifier
            .height(height)
            .clip(RoundedCornerShape(height / 2))
            .background(trackColor)
    ) {
        Box(
            modifier = Modifier
                .fillMaxHeight()
                .fillMaxWidth(fraction = clampedValue / 100f)
                .clip(RoundedCornerShape(height / 2))
                .background(fillColor)
        )
    }
}
package com.pudimproductivity.widget

import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.appwidget.cornerRadius
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Column
import androidx.glance.layout.ColumnScope
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.padding
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import com.pudimproductivity.i18n.Localization

/**
 * Corner radius of the widget card (rounds the background on Android 12+).
 * Matches the app's card shape (`shapes.large` = 16.dp in ui/theme/Theme.kt).
 */
internal val WidgetCornerRadius = 16.dp

/** Padding the card applies on each side. Used to compute the inner width. */
internal val CardHorizontalPadding = 12.dp

/**
 * Shared scaffolding for the Habits widget. Glance 1.1.1 has no `weight`, no
 * `Arrangement`, no fraction-fill and no clipping, so the layout uses fixed
 * spacing and simple rows.
 */
@Composable
internal fun WidgetCard(content: @Composable ColumnScope.() -> Unit) {
    Column(
        modifier = GlanceModifier
            .fillMaxSize()
            .cornerRadius(WidgetCornerRadius)
            .background(GlanceTheme.colors.background)
            .padding(CardHorizontalPadding)
    ) {
        content()
    }
}

/**
 * Streak indicator mirroring the app's StreakBadge (ui/components/StreakBadge.kt):
 * a fire emoji, the current count in the tertiary colour, and an optional
 * muted "(best: N)" suffix.
 */
@Composable
internal fun WidgetStreakBadge(
    count: Int,
    best: Int,
    modifier: GlanceModifier = GlanceModifier
) {
    Row(
        modifier = modifier,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = "🔥",
            style = TextStyle(fontSize = 14.sp)
        )
        Spacer(GlanceModifier.width(2.dp))
        Text(
            text = count.toString(),
            maxLines = 1,
            style = TextStyle(
                color = GlanceTheme.colors.onTertiaryContainer,
                fontWeight = FontWeight.Bold,
                fontSize = 13.sp
            )
        )
        if (best > count) {
            Spacer(GlanceModifier.width(4.dp))
            Text(
                text = Localization.text("streak.best", "count" to best),
                maxLines = 1,
                style = TextStyle(
                    color = GlanceTheme.colors.onSurfaceVariant,
                    fontSize = 11.sp
                )
            )
        }
    }
}

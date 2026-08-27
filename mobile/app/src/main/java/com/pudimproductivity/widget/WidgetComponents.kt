package com.pudimproductivity.widget

import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalSize
import androidx.glance.action.Action
import androidx.glance.action.clickable
import androidx.glance.appwidget.CheckBox
import androidx.glance.appwidget.cornerRadius
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.ColumnScope
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxHeight
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.size
import androidx.glance.layout.width
import androidx.glance.semantics.contentDescription
import androidx.glance.semantics.semantics
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextDecoration
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import com.pudimproductivity.i18n.Localization

/**
 * Corner radius of the widget card (rounds the background on Android 12+).
 * Matches the app's card shape (`shapes.large` = 16.dp in ui/theme/Theme.kt).
 */
internal val WidgetCornerRadius = 16.dp

/** Height of the pill-shaped progress bar. */
internal val ProgressBarHeight = 8.dp

/** Padding the card applies on each side — used to compute the inner width. */
internal val CardHorizontalPadding = 12.dp

/** Width of the left progress panel in the 4×2 (and taller) widget layouts. */
internal val WidgetProgressPanelWidth = 64.dp

/** Gap between the progress panel and the task column in the same layouts. */
internal val WidgetPanelGap = 8.dp

/**
 * Shared scaffolding for the Tasks and Habits widgets so both cards stay
 * visually identical. Glance 1.1.1 has no `weight`, no `Arrangement`, no
 * fraction-fill and no clipping, so the layout uses fixed spacing and the
 * progress fill width is derived from [LocalSize] (the widget's current
 * responsive size — always one of the sizes in the widget's `SizeMode.Responsive`).
 */

/** Rounded, padded brand card used as the root of every widget. The background
 *  uses the app's `background` token — warm cream in light, dark navy in dark —
 *  so the card reads as a piece of the app instead of a flat white slab. */
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
 * Card header: bold title, a subtle count, an optional round "+" quick-add and
 * an optional trailing element (e.g. the compact habit ring). The whole row
 * opens the app (deep link) when tapped.
 */
@Composable
internal fun WidgetHeader(
    title: String,
    countText: String,
    onOpen: Action,
    onAdd: Action? = null,
    trailing: (@Composable () -> Unit)? = null,
    showCount: Boolean = true
) {
    Row(
        modifier = GlanceModifier.fillMaxWidth().clickable(onOpen),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = title,
            maxLines = 1,
            style = TextStyle(
                fontWeight = FontWeight.Bold,
                color = GlanceTheme.colors.onBackground
            )
        )
        if (showCount) {
            Spacer(GlanceModifier.width(8.dp))
            Text(
                text = countText,
                maxLines = 1,
                style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
            )
        }
        if (onAdd != null) {
            Spacer(GlanceModifier.width(6.dp))
            WidgetAddButton(onClick = onAdd)
        }
        if (trailing != null) {
            Spacer(GlanceModifier.width(8.dp))
            trailing()
        }
    }
}

/** Small round "+" button that deep-links to the create screen. */
@Composable
internal fun WidgetAddButton(onClick: Action) {
    Box(
        modifier = GlanceModifier
            .size(26.dp)
            .cornerRadius(13.dp)
            .background(GlanceTheme.colors.primary)
            .clickable(onClick)
            .semantics { contentDescription = "Add" },
        contentAlignment = Alignment.Center
    ) {
        Text(
            text = "+",
            style = TextStyle(
                color = GlanceTheme.colors.onPrimary,
                fontWeight = FontWeight.Bold
            )
        )
    }
}

/**
 * Pill-shaped progress bar matching the app's `ProgressBar` (rounded track and
 * fill). Glance 1.1.1 has no fraction-fill, so the fill width is computed from
 * [LocalSize] minus the card padding.
 */
@Composable
internal fun WidgetProgress(
    progress: Float,
    color: ColorProvider,
    modifier: GlanceModifier = GlanceModifier,
    width: Dp? = null,
    height: Dp = ProgressBarHeight
) {
    val fraction = progress.coerceIn(0f, 1f)
    val barWidth = width ?: (LocalSize.current.width - CardHorizontalPadding * 2).coerceAtLeast(0.dp)
    Row(
        modifier = modifier
            .width(barWidth)
            .height(height)
            .cornerRadius(height / 2)
            .background(GlanceTheme.colors.surfaceVariant)
    ) {
        Box(
            modifier = GlanceModifier
                .width(barWidth * fraction)
                .fillMaxHeight()
                .cornerRadius(height / 2)
                .background(color)
        ) { }
    }
}

/**
 * Left-side progress panel: dominant completion number, a small "of N done"
 * label and the thin pill bar. Used by the 4×2 (and taller) widget layouts so
 * progress is readable at a glance without opening the app.
 */
@Composable
internal fun WidgetProgressPanel(
    done: Int,
    total: Int,
    color: ColorProvider,
    ofLabel: String,
    barWidth: Dp = 56.dp,
    modifier: GlanceModifier = GlanceModifier
) {
    Column(modifier = modifier) {
        Text(
            text = done.toString(),
            maxLines = 1,
            style = TextStyle(
                color = GlanceTheme.colors.onSurface,
                fontWeight = FontWeight.Bold,
                fontSize = 28.sp
            )
        )
        if (total > 0) {
            Text(
                text = ofLabel,
                maxLines = 2,
                style = TextStyle(
                    color = GlanceTheme.colors.onSurfaceVariant,
                    fontSize = 10.sp
                )
            )
            Spacer(GlanceModifier.height(4.dp))
            WidgetProgress(
                progress = done / total.toFloat(),
                color = color,
                width = barWidth
            )
        }
    }
}

/** One row in a widget list: checkbox + title + optional trailing badge. */
@Composable
internal fun WidgetCheckRow(
    checked: Boolean,
    onToggle: Action,
    toggleDescription: String,
    title: String,
    onOpen: Action,
    titleDecoration: TextDecoration? = null,
    compact: Boolean = false,
    trailing: (@Composable () -> Unit)? = null
) {
    Row(
        modifier = GlanceModifier
            .fillMaxWidth()
            .padding(vertical = if (compact) 1.dp else 2.dp)
            .clickable(onOpen)
            .semantics { contentDescription = Localization.text("widgets.openInApp", "title" to title) },
        verticalAlignment = Alignment.CenterVertically
    ) {
        CheckBox(
            checked = checked,
            onCheckedChange = onToggle,
            modifier = GlanceModifier.semantics { contentDescription = toggleDescription }
        )
        Spacer(GlanceModifier.width(6.dp))
        Text(
            text = title,
            maxLines = 1,
            style = TextStyle(
                color = GlanceTheme.colors.onSurface,
                textDecoration = titleDecoration
            )
        )
        if (trailing != null) {
            Spacer(GlanceModifier.width(6.dp))
            trailing()
        }
    }
}

/**
 * Streak indicator mirroring the app's StreakBadge (ui/components/StreakBadge.kt):
 * a fire emoji, the current count in the tertiary colour, and an optional
 * muted "(best: N)" suffix — no pill background, exactly like in-app.
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

/** Friendly empty state: emoji + message + one clear compact action pill. */
@Composable
internal fun WidgetEmptyState(
    emoji: String,
    message: String,
    actionLabel: String,
    onAction: Action
) {
    Column(modifier = GlanceModifier.fillMaxWidth().padding(top = 4.dp)) {
        Text(
            text = "$emoji  $message",
            style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
        )
        Spacer(GlanceModifier.height(4.dp))
        Box(
            modifier = GlanceModifier
                .cornerRadius(14.dp)
                .background(GlanceTheme.colors.primary)
                .clickable(onAction)
                .padding(horizontal = 14.dp, vertical = 6.dp),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = actionLabel,
                style = TextStyle(
                    color = GlanceTheme.colors.onPrimary,
                    fontWeight = FontWeight.Bold
                )
            )
        }
    }
}

/** "+N more in the app" — tappable so overflow opens the matching screen. */
@Composable
internal fun OverflowNote(extra: Int, onOpen: Action) {
    if (extra <= 0) return
    Spacer(GlanceModifier.height(2.dp))
    Text(
        text = Localization.text("widgets.moreInApp", "count" to extra),
        style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant),
        modifier = GlanceModifier.clickable(onOpen)
    )
}

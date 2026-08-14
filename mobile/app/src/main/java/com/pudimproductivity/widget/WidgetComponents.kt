package com.pudimproductivity.widget

import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
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

/** Corner radius of the widget card (rounds the background on Android 12+). */
internal val WidgetCornerRadius = 20.dp

/** Height of the pill-shaped progress bar. */
internal val ProgressBarHeight = 8.dp

/** Padding the card applies on each side — used to compute the inner width. */
internal val CardHorizontalPadding = 12.dp

/**
 * Shared scaffolding for the Tasks and Habits widgets so both cards stay
 * visually identical. Glance 1.1.1 has no `weight`, no `Arrangement`, no
 * fraction-fill and no clipping, so the layout uses fixed spacing and the
 * progress fill width is derived from [LocalSize] (the widget's current
 * responsive size — always one of the sizes in the widget's `SizeMode.Responsive`).
 */

/** Rounded, padded brand card used as the root of every widget. */
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
    modifier: GlanceModifier = GlanceModifier
) {
    val fraction = progress.coerceIn(0f, 1f)
    val innerWidth = (LocalSize.current.width - CardHorizontalPadding * 2).coerceAtLeast(0.dp)
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(ProgressBarHeight)
            .cornerRadius(ProgressBarHeight / 2)
            .background(GlanceTheme.colors.surfaceVariant)
    ) {
        Box(
            modifier = GlanceModifier
                .width(innerWidth * fraction)
                .fillMaxHeight()
                .cornerRadius(ProgressBarHeight / 2)
                .background(color)
        ) { }
    }
}

/**
 * Compact donut-style ring showing the habit completion count, used by the
 * Habits widget's small layout. Glance 1.1.1 has no determinate circular
 * progress indicator, so the ring is a track-coloured disc with a
 * card-coloured hole and the "done/total" count in the centre.
 */
@Composable
internal fun WidgetProgressRing(
    done: Int,
    scheduled: Int,
    color: ColorProvider
) {
    Box(
        modifier = GlanceModifier
            .size(36.dp)
            .cornerRadius(18.dp)
            .background(if (scheduled > 0) color else GlanceTheme.colors.surfaceVariant),
        contentAlignment = Alignment.Center
    ) {
        // Card-coloured hole — both children are centred, so this reads as a ring.
        Box(
            modifier = GlanceModifier
                .size(24.dp)
                .cornerRadius(12.dp)
                .background(GlanceTheme.colors.background)
        ) { }
        Text(
            text = "$done/$scheduled",
            style = TextStyle(
                color = GlanceTheme.colors.onSurface,
                fontWeight = FontWeight.Bold
            )
        )
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

/** Rounded pill used for streak counts (matches the app's StreakBadge accent). */
@Composable
internal fun WidgetBadge(text: String, modifier: GlanceModifier = GlanceModifier) {
    Text(
        text = text,
        maxLines = 1,
        modifier = modifier
            .cornerRadius(8.dp)
            .background(GlanceTheme.colors.primaryContainer)
            .padding(horizontal = 6.dp, vertical = 2.dp),
        style = TextStyle(
            color = GlanceTheme.colors.onPrimaryContainer,
            fontWeight = FontWeight.Bold
        )
    )
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

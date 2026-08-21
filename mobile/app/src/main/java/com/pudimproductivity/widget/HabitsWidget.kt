package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.ExperimentalGlanceApi
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalSize
import androidx.glance.action.clickable
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.layout.Alignment
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import com.pudimproductivity.i18n.Localization
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Responsive breakpoints for the Habits widget — same shape as the Tasks
 * widget so both cards look consistent side by side.
 */
private val HABITS_SIZES = setOf(
    DpSize(250.dp, 50.dp),   // 4×1 compact strip
    DpSize(110.dp, 110.dp),  // 2×2 small tile
    DpSize(180.dp, 110.dp),  // narrow fallback
    DpSize(250.dp, 110.dp),  // 4×2 primary
    DpSize(250.dp, 180.dp),
    DpSize(250.dp, 250.dp)
)

private const val MAX_HABIT_ROWS_NARROW = 2
private const val MAX_HABIT_ROWS_PANEL = 4
private const val MAX_HABIT_ROWS_LARGE = 8

// Matches the app's ProgressVariant.HABIT fill color (ui/components/ProgressBar.kt).
private val HabitGreen = ColorProvider(Color(0xFF10B981))

/**
 * "Today's Habits" home-screen widget: habits scheduled today with a quick
 * check-off for today's completion. Data comes from the offline-first local
 * SQLite DB via [WidgetData].
 *
 * The card adapts to its size: 4×1 shows a one-line strip, 2×2 a big-number
 * tile, the 4×2 primary layout shows a left progress panel (habit green) with
 * up to four habit rows (with streak badges), and tall widgets use a
 * scrollable [LazyColumn].
 */
object HabitsWidget : GlanceAppWidget() {

    override val sizeMode: SizeMode = SizeMode.Responsive(HABITS_SIZES)

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        Localization.init(context)
        val snapshot = withContext(Dispatchers.IO) { WidgetData.loadHabits(context) }
        provideContent {
            GlanceTheme(colors = WidgetColors.providers) {
                HabitsContent(snapshot)
            }
        }
    }
}

@Composable
private fun HabitsContent(snapshot: HabitsSnapshot) {
    // The Glance 1.1.1 runtime wraps provideContent in its own per-size
    // rendering pass (SizeMode.Responsive on HabitsWidget), so LocalSize
    // reflects the current candidate size and the card just branches on it.
    HabitsCard(snapshot)
}

@Composable
private fun HabitsCard(snapshot: HabitsSnapshot) {
    val size = LocalSize.current
    if (size.height <= 60.dp) {
        // 4×1 strip: one line, no rows.
        WidgetCard { HabitsStrip(snapshot) }
        return
    }
    if (size.width < 140.dp) {
        // 2×2 tile: big number, no rows.
        WidgetCard { HabitsSmall(snapshot) }
        return
    }
    WidgetCard {
        HabitsHeader(snapshot, showRing = size.width < 200.dp)
        if (size.width < 200.dp) {
            HabitsNarrow(snapshot)
        } else {
            HabitsWide(snapshot, size.height)
        }
    }
}

/** 4×1: title + thin progress bar + "done/total", in a single row. */
@Composable
private fun HabitsStrip(snapshot: HabitsSnapshot) {
    Row(
        modifier = GlanceModifier.fillMaxWidth().clickable(openHabitsAction()),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = Localization.text("widgets.habits.title"),
            maxLines = 1,
            style = TextStyle(fontWeight = FontWeight.Bold, color = GlanceTheme.colors.onBackground)
        )
        if (snapshot.scheduledToday > 0) {
            Spacer(GlanceModifier.width(8.dp))
            WidgetProgress(
                progress = snapshot.doneToday / snapshot.scheduledToday.toFloat(),
                color = HabitGreen,
                width = 56.dp
            )
        }
        Spacer(GlanceModifier.width(8.dp))
        Text(
            text = if (snapshot.scheduledToday > 0) "${snapshot.doneToday}/${snapshot.scheduledToday}" else Localization.text("widgets.habits.noneToday"),
            maxLines = 1,
            style = TextStyle(color = GlanceTheme.colors.primary, fontWeight = FontWeight.Bold, fontSize = 13.sp)
        )
    }
}

/** 2×2: dominant completion number + "of N done" + thin bar. */
@Composable
private fun HabitsSmall(snapshot: HabitsSnapshot) {
    if (snapshot.scheduledToday == 0) {
        HabitsEmptyState()
        return
    }
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = GlanceModifier.fillMaxWidth().padding(top = 2.dp).clickable(openHabitsAction())
    ) {
        Text(
            text = snapshot.doneToday.toString(),
            maxLines = 1,
            style = TextStyle(color = GlanceTheme.colors.onSurface, fontWeight = FontWeight.Bold, fontSize = 28.sp)
        )
        Text(
            text = doneOfLabel(snapshot),
            maxLines = 1,
            style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant, fontSize = 10.sp)
        )
        Spacer(GlanceModifier.height(3.dp))
        WidgetProgress(
            progress = snapshot.doneToday / snapshot.scheduledToday.toFloat(),
            color = HabitGreen,
            width = 60.dp
        )
    }
}

/** Narrow (~180×110): full-width bar + up to two compact rows. */
@Composable
private fun HabitsNarrow(snapshot: HabitsSnapshot) {
    if (snapshot.habits.isEmpty()) {
        HabitsEmptyState()
        return
    }
    if (snapshot.scheduledToday > 0) {
        Spacer(GlanceModifier.height(4.dp))
        WidgetProgress(progress = snapshot.doneToday / snapshot.scheduledToday.toFloat(), color = HabitGreen)
        Spacer(GlanceModifier.height(6.dp))
    }
    for (row in snapshot.habits.take(MAX_HABIT_ROWS_NARROW)) {
        HabitRowItem(row, compact = true)
    }
    OverflowNote(snapshot.habits.size - MAX_HABIT_ROWS_NARROW, onOpen = openHabitsAction())
}

/** 4×2 (and taller): left progress panel + up to four (or a scrollable) rows. */
@OptIn(ExperimentalGlanceApi::class)
@Composable
private fun HabitsWide(snapshot: HabitsSnapshot, height: Dp) {
    if (snapshot.habits.isEmpty()) {
        HabitsEmptyState()
        return
    }
    val innerWidth = (LocalSize.current.width - CardHorizontalPadding * 2).coerceAtLeast(0.dp)
    val rightWidth = (innerWidth - WidgetProgressPanelWidth - WidgetPanelGap).coerceAtLeast(0.dp)
    Row(modifier = GlanceModifier.fillMaxWidth()) {
        Column(modifier = GlanceModifier.width(WidgetProgressPanelWidth).padding(top = 2.dp)) {
            WidgetProgressPanel(
                done = snapshot.doneToday,
                total = snapshot.scheduledToday,
                color = HabitGreen,
                ofLabel = doneOfLabel(snapshot),
                barWidth = WidgetProgressPanelWidth - 8.dp
            )
        }
        Spacer(GlanceModifier.width(WidgetPanelGap))
        Column(modifier = GlanceModifier.width(rightWidth)) {
            val maxRows = if (height >= 140.dp) MAX_HABIT_ROWS_LARGE else MAX_HABIT_ROWS_PANEL
            if (maxRows == MAX_HABIT_ROWS_LARGE) {
                // Tall layout: LazyColumn scrolls within the fixed widget bounds.
                LazyColumn(
                    modifier = GlanceModifier.fillMaxWidth(),
                    horizontalAlignment = Alignment.Start
                ) {
                    items(snapshot.habits.take(maxRows)) { row ->
                        HabitRowItem(row, compact = true)
                    }
                }
            } else {
                for (row in snapshot.habits.take(maxRows)) {
                    HabitRowItem(row, compact = true)
                }
            }
            OverflowNote(snapshot.habits.size - maxRows, onOpen = openHabitsAction())
        }
    }
}

@Composable
private fun doneOfLabel(snapshot: HabitsSnapshot): String =
    Localization.text("widgets.habits.doneOfScheduled", "done" to snapshot.doneToday, "total" to snapshot.scheduledToday)

/** Friendly empty state with an open-habits CTA. */
@Composable
private fun HabitsEmptyState() {
    WidgetEmptyState(
        emoji = "🌱",
        message = Localization.text("widgets.habits.empty"),
        actionLabel = Localization.text("widgets.habits.open"),
        onAction = openHabitsAction()
    )
}

@Composable
private fun HabitsHeader(snapshot: HabitsSnapshot, showRing: Boolean) {
    // The ring itself shows "done/scheduled", so in compact layouts the count
    // text is redundant — hide it to leave room for the ring in narrow widgets.
    val ringShown = showRing && snapshot.scheduledToday > 0
    val countText = when {
        snapshot.scheduledToday == 0 -> Localization.text("widgets.habits.noneToday")
        else -> "${snapshot.doneToday}/${snapshot.scheduledToday}"
    }
    WidgetHeader(
        title = Localization.text("widgets.habits.title"),
        countText = countText,
        onOpen = openHabitsAction(),
        showCount = !ringShown,
        trailing = if (ringShown) {
            { WidgetProgressRing(snapshot.doneToday, snapshot.scheduledToday, HabitGreen) }
        } else {
            null
        }
    )
}

@Composable
private fun HabitRowItem(row: HabitRow, compact: Boolean) {
    WidgetCheckRow(
        checked = row.completedToday,
        onToggle = toggleHabitAction(row.id),
        toggleDescription = if (row.completedToday) {
            Localization.text("widgets.habits.uncomplete", "title" to row.title)
        } else {
            Localization.text("widgets.habits.complete", "title" to row.title)
        },
        title = row.title,
        onOpen = openHabitsAction(),
        compact = compact
    ) {
        if (row.streak > 0) {
            val badge = if (row.bestStreak > row.streak) {
                "🔥 ${row.streak} · " + Localization.text("streak.best", "count" to row.bestStreak)
            } else {
                "🔥 ${row.streak}"
            }
            WidgetBadge(text = badge)
        }
    }
}


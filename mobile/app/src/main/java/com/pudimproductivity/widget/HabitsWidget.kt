package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.glance.ExperimentalGlanceApi
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalSize
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.layout.Alignment
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.unit.ColorProvider
import com.pudimproductivity.i18n.Localization
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Responsive breakpoints for the Habits widget — same shape as the Tasks
 * widget so both cards look consistent side by side.
 */
private val HABITS_SIZES = setOf(
    DpSize(180.dp, 110.dp),
    DpSize(250.dp, 110.dp),
    DpSize(250.dp, 180.dp),
    DpSize(250.dp, 250.dp)
)

private const val MAX_HABIT_ROWS_COMPACT = 2
private const val MAX_HABIT_ROWS = 5
private const val MAX_HABIT_ROWS_LARGE = 8

// Matches the app's ProgressVariant.HABIT fill color (ui/components/ProgressBar.kt).
private val HabitGreen = ColorProvider(Color(0xFF10B981))

/**
 * "Today's Habits" home-screen widget: habits scheduled today with a quick
 * check-off for today's completion. Data comes from the offline-first local
 * SQLite DB via [WidgetData].
 *
 * The card adapts to its size: the compact layout swaps the progress bar for
 * the [WidgetProgressRing] donut in the header, larger layouts keep the pill
 * bar and show more rows (tall widgets use a scrollable [LazyColumn]).
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
    val compact = size.height < 140.dp

    WidgetCard {
        HabitsHeader(snapshot, showRing = compact)

        if (compact) {
            Spacer(GlanceModifier.height(4.dp))
        } else if (snapshot.scheduledToday > 0) {
            // No point showing an empty progress bar when nothing is scheduled.
            Spacer(GlanceModifier.height(6.dp))
            WidgetProgress(
                progress = snapshot.doneToday / snapshot.scheduledToday.toFloat(),
                color = HabitGreen
            )
            Spacer(GlanceModifier.height(8.dp))
        }

        HabitsBody(snapshot, size.height, compact)
    }
}

@OptIn(ExperimentalGlanceApi::class)
@Composable
private fun HabitsBody(snapshot: HabitsSnapshot, height: Dp, compact: Boolean) {
    if (snapshot.habits.isEmpty()) {
        WidgetEmptyState(
            emoji = "🌱",
            message = Localization.text("widgets.habits.empty"),
            actionLabel = Localization.text("widgets.habits.open"),
            onAction = openHabitsAction()
        )
        return
    }

    val maxRows = when {
        height >= 200.dp -> MAX_HABIT_ROWS_LARGE
        height >= 140.dp -> MAX_HABIT_ROWS
        else -> MAX_HABIT_ROWS_COMPACT
    }

    if (maxRows == MAX_HABIT_ROWS_LARGE) {
        LazyColumn(
            modifier = GlanceModifier.fillMaxWidth(),
            horizontalAlignment = Alignment.Start
        ) {
            items(snapshot.habits.take(maxRows)) { row ->
                HabitRowItem(row, compact = compact)
            }
        }
    } else {
        for (row in snapshot.habits.take(maxRows)) {
            HabitRowItem(row, compact = compact)
        }
    }
    OverflowNote(snapshot.habits.size - maxRows, onOpen = openHabitsAction())
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


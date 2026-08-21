package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.ExperimentalGlanceApi
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalSize
import androidx.glance.action.actionParametersOf
import androidx.glance.action.actionStartActivity
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
import androidx.glance.text.TextDecoration
import androidx.glance.text.TextStyle
import com.pudimproductivity.MainActivity
import com.pudimproductivity.i18n.Localization
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Responsive breakpoints for the Tasks widget. On Android 12+ the launcher
 * picks the closest layout for the actual widget size; on older versions the
 * best match per orientation is used (see Glance `SizeMode.Responsive`).
 */
private val TASKS_SIZES = setOf(
    DpSize(250.dp, 50.dp),   // 4×1 compact strip
    DpSize(110.dp, 110.dp),  // 2×2 small tile
    DpSize(180.dp, 110.dp),  // narrow fallback
    DpSize(250.dp, 110.dp),  // 4×2 primary
    DpSize(250.dp, 180.dp),
    DpSize(250.dp, 250.dp)
)

private const val MAX_TASK_ROWS_NARROW = 2
private const val MAX_TASK_ROWS_PANEL = 4
private const val MAX_TASK_ROWS_LARGE = 8

/**
 * "Today's Tasks" home-screen widget: pending one-off tasks with quick
 * check-off. Data comes from the offline-first local SQLite DB via [WidgetData].
 *
 * The card adapts to its size: 4×1 shows a one-line strip, 2×2 a big-number
 * tile, the 4×2 primary layout shows a left progress panel with up to four
 * task rows (done rows stay visible with a restrained strike), and tall
 * widgets use a scrollable [LazyColumn]. Rows, header, "+" and the "+N more"
 * note all deep-link into the app (see [WidgetActions]).
 */
object TasksWidget : GlanceAppWidget() {

    override val sizeMode: SizeMode = SizeMode.Responsive(TASKS_SIZES)

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        Localization.init(context)
        val snapshot = withContext(Dispatchers.IO) { WidgetData.loadTasks(context) }
        provideContent {
            GlanceTheme(colors = WidgetColors.providers) {
                TasksContent(snapshot)
            }
        }
    }
}

@Composable
private fun TasksContent(snapshot: TasksSnapshot) {
    // The Glance 1.1.1 runtime wraps provideContent in its own per-size
    // rendering pass (SizeMode.Responsive on TasksWidget), so LocalSize
    // reflects the current candidate size and the card just branches on it.
    TasksCard(snapshot)
}

@Composable
private fun TasksCard(snapshot: TasksSnapshot) {
    val size = LocalSize.current
    if (size.height <= 60.dp) {
        // 4×1 strip: one line, no rows.
        WidgetCard { TasksStrip(snapshot) }
        return
    }
    if (size.width < 140.dp) {
        // 2×2 tile: big number, no rows.
        WidgetCard { TasksSmall(snapshot) }
        return
    }
    WidgetCard {
        TasksHeader(snapshot)
        if (size.width < 200.dp) {
            TasksNarrow(snapshot)
        } else {
            TasksWide(snapshot, size.height)
        }
    }
}

/** 4×1: title + thin progress bar + "done/total", in a single row. */
@Composable
private fun TasksStrip(snapshot: TasksSnapshot) {
    Row(
        modifier = GlanceModifier.fillMaxWidth().clickable(openTasksAction()),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = Localization.text("widgets.tasks.title"),
            maxLines = 1,
            style = TextStyle(fontWeight = FontWeight.Bold, color = GlanceTheme.colors.onBackground)
        )
        if (snapshot.total > 0) {
            Spacer(GlanceModifier.width(8.dp))
            WidgetProgress(
                progress = snapshot.done / snapshot.total.toFloat(),
                color = GlanceTheme.colors.primary,
                width = 56.dp
            )
        }
        Spacer(GlanceModifier.width(8.dp))
        Text(
            text = if (snapshot.total > 0) "${snapshot.done}/${snapshot.total}" else Localization.text("widgets.tasks.noTasksShort"),
            maxLines = 1,
            style = TextStyle(color = GlanceTheme.colors.primary, fontWeight = FontWeight.Bold, fontSize = 13.sp)
        )
    }
}

/** 2×2: dominant completion number + "of N done" + thin bar. */
@Composable
private fun TasksSmall(snapshot: TasksSnapshot) {
    if (snapshot.total == 0) {
        TasksEmptyState(snapshot)
        return
    }
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = GlanceModifier.fillMaxWidth().padding(top = 2.dp).clickable(openTasksAction())
    ) {
        Text(
            text = snapshot.done.toString(),
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
            progress = snapshot.done / snapshot.total.toFloat(),
            color = GlanceTheme.colors.primary,
            width = 60.dp
        )
    }
}

/** Narrow (~180×110): full-width bar + up to two compact rows. */
@Composable
private fun TasksNarrow(snapshot: TasksSnapshot) {
    if (snapshot.pending.isEmpty()) {
        TasksEmptyState(snapshot)
        return
    }
    if (snapshot.total > 0) {
        Spacer(GlanceModifier.height(4.dp))
        WidgetProgress(progress = snapshot.done / snapshot.total.toFloat(), color = GlanceTheme.colors.primary)
        Spacer(GlanceModifier.height(6.dp))
    }
    for (row in snapshot.visible.take(MAX_TASK_ROWS_NARROW)) {
        TaskRowItem(row, compact = true)
    }
    OverflowNote(snapshot.visible.size - MAX_TASK_ROWS_NARROW, onOpen = openTasksAction())
}

/** 4×2 (and taller): left progress panel + up to four (or a scrollable) rows. */
@OptIn(ExperimentalGlanceApi::class)
@Composable
private fun TasksWide(snapshot: TasksSnapshot, height: Dp) {
    if (snapshot.pending.isEmpty()) {
        TasksEmptyState(snapshot)
        return
    }
    val innerWidth = (LocalSize.current.width - CardHorizontalPadding * 2).coerceAtLeast(0.dp)
    val rightWidth = (innerWidth - WidgetProgressPanelWidth - WidgetPanelGap).coerceAtLeast(0.dp)
    Row(modifier = GlanceModifier.fillMaxWidth()) {
        Column(modifier = GlanceModifier.width(WidgetProgressPanelWidth).padding(top = 2.dp)) {
            WidgetProgressPanel(
                done = snapshot.done,
                total = snapshot.total,
                color = GlanceTheme.colors.primary,
                ofLabel = doneOfLabel(snapshot),
                barWidth = WidgetProgressPanelWidth - 8.dp
            )
        }
        Spacer(GlanceModifier.width(WidgetPanelGap))
        Column(modifier = GlanceModifier.width(rightWidth)) {
            val maxRows = if (height >= 140.dp) MAX_TASK_ROWS_LARGE else MAX_TASK_ROWS_PANEL
            if (maxRows == MAX_TASK_ROWS_LARGE) {
                // Tall layout: LazyColumn scrolls within the fixed widget bounds.
                LazyColumn(
                    modifier = GlanceModifier.fillMaxWidth(),
                    horizontalAlignment = Alignment.Start
                ) {
                    items(snapshot.visible.take(maxRows)) { row ->
                        TaskRowItem(row, compact = true)
                    }
                }
            } else {
                for (row in snapshot.visible.take(maxRows)) {
                    TaskRowItem(row, compact = true)
                }
            }
            OverflowNote(snapshot.visible.size - maxRows, onOpen = openTasksAction())
        }
    }
}

@Composable
private fun doneOfLabel(snapshot: TasksSnapshot): String =
    Localization.text("widgets.tasks.doneOfTotal", "done" to snapshot.done, "total" to snapshot.total)

/** Friendly empty/complete state with a single add-task CTA. */
@Composable
private fun TasksEmptyState(snapshot: TasksSnapshot) {
    WidgetEmptyState(
        emoji = if (snapshot.total == 0) "🎯" else "🎉",
        message = if (snapshot.total == 0) Localization.text("widgets.tasks.empty") else Localization.text("widgets.tasks.allDone"),
        actionLabel = Localization.text("widgets.tasks.add"),
        onAction = actionStartActivity<MainActivity>(
            actionParametersOf(EXTRA_SCREEN_KEY to MainActivity.SCREEN_TASK_CREATE)
        )
    )
}

@Composable
private fun TasksHeader(snapshot: TasksSnapshot) {
    val countText = when {
        snapshot.total == 0 -> Localization.text("widgets.tasks.noTasksShort")
        snapshot.remaining == 0 -> Localization.text("widgets.tasks.allDoneShort")
        else -> Localization.text("widgets.tasks.remaining", "count" to snapshot.remaining)
    }
    WidgetHeader(
        title = Localization.text("widgets.tasks.title"),
        countText = countText,
        onOpen = openTasksAction(),
        onAdd = actionStartActivity<MainActivity>(
            actionParametersOf(EXTRA_SCREEN_KEY to MainActivity.SCREEN_TASK_CREATE)
        )
    )
}

@Composable
private fun TaskRowItem(row: TaskRow, compact: Boolean) {
    WidgetCheckRow(
        checked = row.done,
        onToggle = toggleTaskAction(row.id, done = !row.done),
        toggleDescription = if (row.done) {
            Localization.text("widgets.tasks.markNotDone", "title" to row.title)
        } else {
            Localization.text("widgets.tasks.markDone", "title" to row.title)
        },
        title = row.title,
        onOpen = actionStartActivity<MainActivity>(
            actionParametersOf(
                EXTRA_SCREEN_KEY to MainActivity.SCREEN_TASK_DETAIL,
                EXTRA_TASK_ID_KEY to row.id
            )
        ),
        titleDecoration = if (row.done) TextDecoration.LineThrough else null,
        compact = compact
    )
}


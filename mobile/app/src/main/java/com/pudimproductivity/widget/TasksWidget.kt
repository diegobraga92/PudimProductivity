package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.glance.ExperimentalGlanceApi
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalSize
import androidx.glance.action.actionParametersOf
import androidx.glance.action.actionStartActivity
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.layout.Alignment
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.text.TextDecoration
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
    DpSize(180.dp, 110.dp),
    DpSize(250.dp, 110.dp),
    DpSize(250.dp, 180.dp),
    DpSize(250.dp, 250.dp)
)

private const val MAX_TASK_ROWS_COMPACT = 2
private const val MAX_TASK_ROWS = 5
private const val MAX_TASK_ROWS_LARGE = 8

/**
 * "Today's Tasks" home-screen widget: pending one-off tasks with quick
 * check-off. Data comes from the offline-first local SQLite DB via [WidgetData].
 *
 * The card adapts to its size: compact widgets show fewer rows with a thin
 * layout, tall widgets use a scrollable [LazyColumn]. Rows, header, "+" and
 * the "+N more" note all deep-link into the app (see [WidgetActions]).
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
    WidgetCard {
        TasksHeader(snapshot)

        // No point showing an empty progress bar before the first task exists.
        if (snapshot.total > 0) {
            Spacer(GlanceModifier.height(6.dp))
            WidgetProgress(
                progress = snapshot.done / snapshot.total.toFloat(),
                color = GlanceTheme.colors.primary
            )
            Spacer(GlanceModifier.height(8.dp))
        }

        TasksBody(snapshot, size.height)
    }
}

@OptIn(ExperimentalGlanceApi::class)
@Composable
private fun TasksBody(snapshot: TasksSnapshot, height: Dp) {
    if (snapshot.pending.isEmpty()) {
        WidgetEmptyState(
            emoji = if (snapshot.total == 0) "🎯" else "🎉",
            message = if (snapshot.total == 0) Localization.text("widgets.tasks.empty") else Localization.text("widgets.tasks.allDone"),
            actionLabel = Localization.text("widgets.tasks.add"),
            onAction = actionStartActivity<MainActivity>(
                actionParametersOf(EXTRA_SCREEN_KEY to MainActivity.SCREEN_TASK_CREATE)
            )
        )
        return
    }

    val maxRows = when {
        height >= 200.dp -> MAX_TASK_ROWS_LARGE
        height >= 140.dp -> MAX_TASK_ROWS
        else -> MAX_TASK_ROWS_COMPACT
    }
    val compact = height < 140.dp

    if (maxRows == MAX_TASK_ROWS_LARGE) {
        // Tall layout: LazyColumn scrolls within the fixed widget bounds.
        LazyColumn(
            modifier = GlanceModifier.fillMaxWidth(),
            horizontalAlignment = Alignment.Start
        ) {
            items(snapshot.pending.take(maxRows)) { row ->
                TaskRowItem(row, compact = compact)
            }
        }
    } else {
        for (row in snapshot.pending.take(maxRows)) {
            TaskRowItem(row, compact = compact)
        }
    }
    OverflowNote(snapshot.pending.size - maxRows, onOpen = openTasksAction())
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


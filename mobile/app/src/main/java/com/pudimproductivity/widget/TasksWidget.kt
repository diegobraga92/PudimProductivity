package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.dp
import androidx.glance.Button
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.action.actionParametersOf
import androidx.glance.action.actionStartActivity
import androidx.glance.action.clickable
import androidx.glance.appwidget.CheckBox
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.LinearProgressIndicator
import androidx.glance.appwidget.provideContent
import androidx.glance.background
import androidx.glance.layout.Alignment
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
import androidx.glance.layout.fillMaxSize
import androidx.glance.layout.fillMaxWidth
import androidx.glance.layout.height
import androidx.glance.layout.padding
import androidx.glance.layout.width
import androidx.glance.text.FontWeight
import androidx.glance.text.Text
import androidx.glance.text.TextDecoration
import androidx.glance.text.TextStyle
import com.pudimproductivity.MainActivity
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

private const val MAX_TASK_ROWS = 5

/**
 * "Today's Tasks" home-screen widget (Phase 10): pending one-off tasks with
 * quick check-off. Rendered with Jetpack Glance 1.1.1; data comes from the
 * offline-first local SQLite DB via [WidgetData].
 *
 * Note: Glance 1.1 has no Compose-style `weight`, so rows use fixed spacing
 * and `maxLines` truncation instead.
 */
object TasksWidget : GlanceAppWidget() {

    override suspend fun provideGlance(context: Context, id: GlanceId) {
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
    Column(
        modifier = GlanceModifier
            .fillMaxSize()
            .background(GlanceTheme.colors.background)
            .padding(12.dp)
    ) {
        Row(
            modifier = GlanceModifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "Today's Tasks",
                style = TextStyle(fontWeight = FontWeight.Bold, color = GlanceTheme.colors.onBackground)
            )
            Spacer(GlanceModifier.width(8.dp))
            Text(
                text = "${snapshot.done}/${snapshot.total}",
                style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
            )
        }

        Spacer(GlanceModifier.height(6.dp))

        LinearProgressIndicator(
            progress = if (snapshot.total > 0) snapshot.done / snapshot.total.toFloat() else 0f,
            modifier = GlanceModifier.fillMaxWidth(),
            color = GlanceTheme.colors.primary,
            backgroundColor = GlanceTheme.colors.surfaceVariant
        )

        Spacer(GlanceModifier.height(8.dp))

        if (snapshot.pending.isEmpty()) {
            Text(
                text = if (snapshot.total == 0) "No tasks yet — add one!" else "All done for today 🎉",
                style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
            )
            Spacer(GlanceModifier.height(10.dp))
            Button(
                text = "Add task",
                onClick = actionStartActivity<MainActivity>(
                    actionParametersOf(EXTRA_SCREEN_KEY to MainActivity.SCREEN_TASK_CREATE)
                )
            )
        } else {
            for (row in snapshot.pending.take(MAX_TASK_ROWS)) {
                TaskRowItem(row)
            }
            if (snapshot.pending.size > MAX_TASK_ROWS) {
                Spacer(GlanceModifier.height(2.dp))
                Text(
                    text = "+${snapshot.pending.size - MAX_TASK_ROWS} more in the app",
                    style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
                )
            }
        }
    }
}

@Composable
private fun TaskRowItem(row: TaskRow) {
    Row(
        modifier = GlanceModifier.fillMaxWidth().padding(vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        CheckBox(
            checked = row.done,
            onCheckedChange = toggleTaskAction(row.id, done = !row.done),
            modifier = GlanceModifier.width(40.dp)
        )
        Text(
            text = row.title,
            maxLines = 1,
            style = TextStyle(
                color = GlanceTheme.colors.onSurface,
                textDecoration = if (row.done) TextDecoration.LineThrough else null
            ),
            modifier = GlanceModifier.clickable(
                actionStartActivity<MainActivity>(
                    actionParametersOf(
                        EXTRA_SCREEN_KEY to MainActivity.SCREEN_TASK_DETAIL,
                        EXTRA_TASK_ID_KEY to row.id
                    )
                )
            )
        )
    }
}


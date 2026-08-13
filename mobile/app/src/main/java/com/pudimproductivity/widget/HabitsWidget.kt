package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
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
import androidx.glance.text.TextStyle
import androidx.glance.unit.ColorProvider
import com.pudimproductivity.MainActivity
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

private const val MAX_HABIT_ROWS = 5

// Matches the app's ProgressVariant.HABIT fill color (ui/components/ProgressBar.kt).
private val HabitGreen = ColorProvider(Color(0xFF10B981))

/**
 * "Today's Habits" home-screen widget (Phase 10): habits scheduled today with
 * a quick check-off for today's completion. Data comes from the offline-first
 * local SQLite DB via [WidgetData].
 */
object HabitsWidget : GlanceAppWidget() {

    override suspend fun provideGlance(context: Context, id: GlanceId) {
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
                text = "Today's Habits",
                style = TextStyle(fontWeight = FontWeight.Bold, color = GlanceTheme.colors.onBackground)
            )
            Spacer(GlanceModifier.width(8.dp))
            Text(
                text = "${snapshot.doneToday}/${snapshot.scheduledToday}",
                style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
            )
        }

        Spacer(GlanceModifier.height(6.dp))

        LinearProgressIndicator(
            progress = if (snapshot.scheduledToday > 0) {
                snapshot.doneToday / snapshot.scheduledToday.toFloat()
            } else 0f,
            modifier = GlanceModifier.fillMaxWidth(),
            color = HabitGreen,
            backgroundColor = GlanceTheme.colors.surfaceVariant
        )

        Spacer(GlanceModifier.height(8.dp))

        if (snapshot.habits.isEmpty()) {
            Text(
                text = "No habits scheduled today",
                style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
            )
            Spacer(GlanceModifier.height(10.dp))
            androidx.glance.Button(
                text = "Open habits",
                onClick = actionStartActivity<MainActivity>(
                    actionParametersOf(EXTRA_SCREEN_KEY to MainActivity.SCREEN_HABITS)
                )
            )
        } else {
            for (row in snapshot.habits.take(MAX_HABIT_ROWS)) {
                HabitRowItem(row)
            }
            if (snapshot.habits.size > MAX_HABIT_ROWS) {
                Spacer(GlanceModifier.height(2.dp))
                Text(
                    text = "+${snapshot.habits.size - MAX_HABIT_ROWS} more in the app",
                    style = TextStyle(color = GlanceTheme.colors.onSurfaceVariant)
                )
            }
        }
    }
}

@Composable
private fun HabitRowItem(row: HabitRow) {
    Row(
        modifier = GlanceModifier.fillMaxWidth().padding(vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        CheckBox(
            checked = row.completedToday,
            onCheckedChange = toggleHabitAction(row.id),
            modifier = GlanceModifier.width(40.dp)
        )
        Text(
            text = row.title,
            maxLines = 1,
            style = TextStyle(color = GlanceTheme.colors.onSurface),
            modifier = GlanceModifier.clickable(
                actionStartActivity<MainActivity>(
                    actionParametersOf(EXTRA_SCREEN_KEY to MainActivity.SCREEN_HABITS)
                )
            )
        )
        if (row.streak > 0) {
            Spacer(GlanceModifier.width(6.dp))
            Text(
                text = "🔥${row.streak}",
                style = TextStyle(
                    color = GlanceTheme.colors.primary,
                    fontWeight = FontWeight.Bold
                )
            )
        }
    }
}


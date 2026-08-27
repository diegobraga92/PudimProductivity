package com.pudimproductivity.widget

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.ui.unit.DpSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.glance.ExperimentalGlanceApi
import androidx.glance.GlanceId
import androidx.glance.GlanceModifier
import androidx.glance.GlanceTheme
import androidx.glance.LocalContext
import androidx.glance.LocalSize
import androidx.glance.background
import androidx.glance.appwidget.CheckBox
import androidx.glance.appwidget.CheckboxDefaults
import androidx.glance.appwidget.GlanceAppWidget
import androidx.glance.appwidget.SizeMode
import androidx.glance.appwidget.cornerRadius
import androidx.glance.appwidget.lazy.LazyColumn
import androidx.glance.appwidget.lazy.items
import androidx.glance.appwidget.provideContent
import androidx.glance.layout.Alignment
import androidx.glance.layout.Box
import androidx.glance.layout.Column
import androidx.glance.layout.Row
import androidx.glance.layout.Spacer
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
import androidx.glance.text.TextStyle
import com.pudimproductivity.i18n.Localization
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * Responsive breakpoints for the Habits widget. Every size renders the same
 * clean layout — a bold heading above a scrollable list of today's habits —
 * so resizing never produces a different-looking card.
 */
private val HABITS_SIZES = setOf(
    DpSize(250.dp, 50.dp),   // 4×1 compact strip (heading only)
    DpSize(110.dp, 110.dp),  // 2×2 small tile
    DpSize(180.dp, 110.dp),  // narrow fallback
    DpSize(250.dp, 110.dp),  // 4×2 primary
    DpSize(250.dp, 180.dp),
    DpSize(250.dp, 250.dp),
    DpSize(250.dp, 330.dp),  // taller stretches — the card lists more habits
    DpSize(250.dp, 440.dp),
    DpSize(250.dp, 550.dp)
)

/**
 * "Habits" home-screen widget: a bold, localized title with a scrollable list
 * of today's habits. The only interaction is checking/unchecking each habit
 * for today; nothing else on the card opens the app. Data comes from the
 * offline-first local SQLite DB via [WidgetData].
 */
object HabitsWidget : GlanceAppWidget() {

    override val sizeMode: SizeMode = SizeMode.Responsive(HABITS_SIZES)

    override suspend fun provideGlance(context: Context, id: GlanceId) {
        Localization.init(context)
        val snapshot = withContext(Dispatchers.IO) { WidgetData.loadHabits(context) }
        provideContent {
            // Follow the app's theme (System / Light / Dark), not just the system.
            GlanceTheme(colors = WidgetColors.resolve(context)) {
                HabitsContent(snapshot)
            }
        }
    }
}

@OptIn(ExperimentalGlanceApi::class)
@Composable
private fun HabitsContent(snapshot: HabitsSnapshot) {
    WidgetCard {
        // On the 4×1 strip there is only room for the heading.
        val compact = LocalSize.current.height <= 60.dp
        if (compact) {
            HabitsTitle(compact = true)
        } else {
            // Root LazyColumn gives the list a bounded height so it actually
            // renders and scrolls inside the card. The heading + segmented
            // progress dots are the first item, then one row per habit.
            LazyColumn(
                modifier = GlanceModifier.fillMaxSize(),
                horizontalAlignment = Alignment.Start
            ) {
                item { HabitsHeader(snapshot) }
                if (snapshot.habits.isEmpty()) {
                    item { HabitsEmptyState() }
                } else {
                    items(snapshot.habits) { row ->
                        HabitRowItem(row)
                    }
                }
            }
        }
    }
}

/**
 * Heading + segmented progress indicator. Rendered as the list's first item —
 * kept as a single Column because Glance lazy items must have exactly one child.
 */
@Composable
private fun HabitsHeader(snapshot: HabitsSnapshot) {
    Column(GlanceModifier.fillMaxWidth()) {
        HabitsTitle(compact = false)
        if (snapshot.scheduledToday > 0) {
            Spacer(GlanceModifier.height(8.dp))
            HabitsProgress(snapshot)
        }
    }
}

/** Bold "Habits" heading matching the app's top-bar title (titleLarge). */
@Composable
private fun HabitsTitle(compact: Boolean) {
    Text(
        text = Localization.text("widgets.habits.title"),
        maxLines = 1,
        style = TextStyle(
            color = GlanceTheme.colors.onBackground,
            fontWeight = FontWeight.Bold,
            fontSize = when {
                compact -> 17.sp
                LocalSize.current.width < 200.dp -> 20.sp
                else -> 22.sp
            }
        )
    )
}

/**
 * Segmented progress indicator: one rounded square per habit (green when
 * completed, outline when pending) plus a "done/total" count. Narrow widgets
 * show only the count because the dots would not fit.
 */
@Composable
private fun HabitsProgress(snapshot: HabitsSnapshot) {
    if (LocalSize.current.width < 200.dp) {
        Text(
            text = "${snapshot.doneToday}/${snapshot.scheduledToday}",
            style = TextStyle(
                color = GlanceTheme.colors.secondary,
                fontWeight = FontWeight.Bold,
                fontSize = 14.sp
            )
        )
        return
    }
    val visible = snapshot.scheduledToday.coerceAtMost(12)
    Row(verticalAlignment = Alignment.CenterVertically) {
        for (i in 0 until visible) {
            Box(
                modifier = GlanceModifier
                    .size(12.dp)
                    .cornerRadius(4.dp)
                    .background(
                        if (snapshot.habits[i].completedToday) GlanceTheme.colors.secondary
                        else GlanceTheme.colors.outline
                    )
            ) { }
            if (i < visible - 1) {
                Spacer(GlanceModifier.width(4.dp))
            }
        }
        if (snapshot.scheduledToday > visible) {
            Spacer(GlanceModifier.width(4.dp))
            Text(
                text = "+${snapshot.scheduledToday - visible}",
                style = TextStyle(
                    color = GlanceTheme.colors.onSurfaceVariant,
                    fontSize = 12.sp
                )
            )
        }
        Spacer(GlanceModifier.width(8.dp))
        Text(
            text = "${snapshot.doneToday}/${snapshot.scheduledToday}",
            style = TextStyle(
                color = GlanceTheme.colors.secondary,
                fontWeight = FontWeight.Bold,
                fontSize = 13.sp
            )
        )
    }
}

/**
 * One habit as a web-style card — mirrors the web's `.card-habit` (white
 * surface, --radius-md 12dp, 1px border). Glance has no border modifier, so
 * the border is a 1dp ring of [WidgetColors.cardBorder] colour drawn behind a
 * surface-coloured card. The checkbox is the only tap target.
 */
@Composable
private fun HabitRowItem(row: HabitRow) {
    // Narrow widgets have no room for the streak next to a long title.
    val showStreak = LocalSize.current.width >= 200.dp
    Box(
        modifier = GlanceModifier
            .fillMaxWidth()
            .padding(vertical = 3.dp)
    ) {
        // Border ring.
        Box(
            modifier = GlanceModifier
                .fillMaxWidth()
                .cornerRadius(12.dp)
                .background(WidgetColors.cardBorder(LocalContext.current))
                .padding(1.dp)
        ) {
            // Surface card.
            Row(
                modifier = GlanceModifier
                    .fillMaxWidth()
                    .cornerRadius(11.dp)
                    .background(GlanceTheme.colors.surface)
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                CheckBox(
                    checked = row.completedToday,
                    onCheckedChange = toggleHabitAction(row.id),
                    // Use the public, @Composable `colors` overload instead of the
                    // @RestrictTo(LIBRARY_GROUP) `checkBoxColors` — the latter is
                    // internal to the androidx.glance library group and fails lint
                    // (RestrictedApi) when called from app code. They behave identically.
                    colors = CheckboxDefaults.colors(
                        checkedColor = GlanceTheme.colors.secondary,
                        uncheckedColor = GlanceTheme.colors.outline
                    ),
                    modifier = GlanceModifier.semantics {
                        contentDescription = if (row.completedToday) {
                            Localization.text("widgets.habits.uncomplete", "title" to row.title)
                        } else {
                            Localization.text("widgets.habits.complete", "title" to row.title)
                        }
                    }
                )
                Spacer(GlanceModifier.width(10.dp))
                Text(
                    text = row.title,
                    maxLines = 2,
                    style = TextStyle(
                        color = GlanceTheme.colors.onSurface,
                        fontSize = 18.sp
                    )
                )
                if (showStreak && row.streak > 0) {
                    Spacer(GlanceModifier.width(8.dp))
                    WidgetStreakBadge(count = row.streak, best = row.bestStreak)
                }
            }
        }
    }
}

/** Plain, non-interactive empty state — the widget only checks/unchecks habits. */
@Composable
private fun HabitsEmptyState() {
    Column(
        modifier = GlanceModifier.fillMaxWidth().padding(top = 10.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = "🌱  " + Localization.text("widgets.habits.empty"),
            style = TextStyle(
                color = GlanceTheme.colors.onSurfaceVariant,
                fontSize = 16.sp
            )
        )
    }
}

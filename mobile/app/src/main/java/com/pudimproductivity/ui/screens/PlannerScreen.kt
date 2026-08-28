package com.pudimproductivity.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.taskService
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.utils.getDate
import kotlinx.coroutines.launch
import java.time.LocalDate

private val DAY_NAMES = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val FULL_DAY_KEYS = mapOf(
    "mon" to "days.monday", "tue" to "days.tuesday", "wed" to "days.wednesday",
    "thu" to "days.thursday", "fri" to "days.friday", "sat" to "days.saturday",
    "sun" to "days.sunday"
)
private val DEFAULT_COLOR = Color(0xFF3B82F6)

// Grid hours: 6AM to 10PM (16 hourly slots).
private const val GRID_START_MINUTES = 6 * 60 // 6AM
private const val HOUR_HEIGHT_DP = 60

private data class HourSlot(val label: String)

private val HOURS = (6..21).map { h ->
    HourSlot(
        label = if (h > 12) "${h - 12}PM" else if (h == 12) "12PM" else "${h}AM"
    )
}

/** Weekday key (mon..sun) for an ISO date. */
private fun weekdayName(isoDate: String): String {
    val date = LocalDate.parse(isoDate)
    return DAY_NAMES[(date.dayOfWeek.value + 6) % 7] // 0 = Mon
}

/** Whether a task is scheduled on the given date (habits recur, one-offs are date-based). */
private fun appliesOn(task: Task, dateStr: String): Boolean {
    val recurrence = task.recurrence_days
    if (!recurrence.isNullOrEmpty()) return weekdayName(dateStr) in recurrence
    return task.scheduled_date == dateStr
}

private fun parseTimeToMinutes(t: String?): Int {
    if (t.isNullOrBlank()) return 0
    val parts = t.split(":")
    val h = parts.getOrNull(0)?.toIntOrNull() ?: 0
    val m = parts.getOrNull(1)?.toIntOrNull() ?: 0
    return h * 60 + m
}

private fun parseHexColor(hex: String?): Color {
    if (hex == null || hex.length < 7) return DEFAULT_COLOR
    return try {
        Color(("FF" + hex.removePrefix("#")).toLong(16))
    } catch (_: Exception) {
        DEFAULT_COLOR
    }
}

/** True when a hex color is light enough to need dark text on top. */
private fun isLightColor(hex: String?): Boolean {
    val color = parseHexColor(hex)
    val r = (color.red * 255).toInt()
    val g = (color.green * 255).toInt()
    val b = (color.blue * 255).toInt()
    return 0.299 * r + 0.587 * g + 0.114 * b > 160
}

private fun timeRangeLabel(task: Task): String {
    val start = task.start_time?.take(5) ?: "--:--"
    val end = task.end_time?.take(5) ?: "--:--"
    return "$start – $end"
}

/**
 * Mobile Planner: one day at a time.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlannerScreen(onBack: () -> Unit, onOpenTask: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    var dayOffset by remember { mutableStateOf(0) }
    var tasks by remember { mutableStateOf<List<Task>>(emptyList()) }
    var completions by remember { mutableStateOf<List<String>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    val dateStr = getDate(dayOffset)

    fun reload() {
        scope.launch {
            loading = true
            error = null
            try {
                tasks = ApiClient.taskService.listScheduledTasks()
                // Habit completions for the visible day (mirrors web's ✓ + strikethrough).
                completions = ApiClient.taskService
                    .getAllTaskCompletions(dateStr, dateStr)
                    .map { it.completed_date }
            } catch (e: Exception) {
                error = e.message ?: Localization.text("mobile.planner.load.failed")
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }
    LaunchedEffect(dayOffset) { reload() }

    val dayTasks = tasks.filter { appliesOn(it, dateStr) }
        .sortedBy { it.start_time ?: "99:99" }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(Localization.text("nav.planner")) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = Localization.text("common.back"))
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
        ) {
            // Day navigation header
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth()
            ) {
                TextButton(onClick = { dayOffset -= 1 }) {
                    Text("← " + Localization.text("week.prev"))
                }
                Column(
                    modifier = Modifier.weight(1f),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(
                        text = if (dayOffset == 0) Localization.text("day.today")
                        else Localization.text(FULL_DAY_KEYS[weekdayName(dateStr)] ?: "days.monday"),
                        style = MaterialTheme.typography.titleSmall,
                        fontWeight = FontWeight.SemiBold
                    )
                    Text(
                        text = dateStr.substring(5).replace("-", "/"),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                TextButton(
                    onClick = { dayOffset += 1 },
                    enabled = dayOffset < 0
                ) {
                    Text(
                        Localization.text("week.next") + " →",
                        color = if (dayOffset < 0) MaterialTheme.colorScheme.primary
                        else MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.38f)
                    )
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            when {
                loading -> {
                    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
                error != null -> {
                    Column(
                        modifier = Modifier.fillMaxSize(),
                        verticalArrangement = Arrangement.Center,
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = Localization.text("common.error") + ": $error",
                            color = MaterialTheme.colorScheme.error,
                            modifier = Modifier.padding(bottom = 8.dp)
                        )
                        Button(onClick = { reload() }) {
                            Text(Localization.text("common.retry"))
                        }
                    }
                }
                else -> {
                    DayGrid(
                        dateStr = dateStr,
                        tasks = dayTasks,
                        completions = completions,
                        onOpenTask = onOpenTask
                    )
                }
            }
        }
    }
}

@Composable
private fun ColumnScope.DayGrid(
    dateStr: String,
    tasks: List<Task>,
    completions: List<String>,
    onOpenTask: (String) -> Unit
) {
    val gridHeightDp = HOURS.size * HOUR_HEIGHT_DP
    val borderColor = MaterialTheme.colorScheme.outlineVariant

    Row(
        modifier = Modifier
            .fillMaxWidth()
            .weight(1f)
            .verticalScroll(rememberScrollState())
    ) {
        // Time labels column
        Column(modifier = Modifier.width(56.dp)) {
            HOURS.forEach { hour ->
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(HOUR_HEIGHT_DP.dp)
                        .border(1.dp, borderColor.copy(alpha = 0.5f)),
                    contentAlignment = Alignment.CenterEnd
                ) {
                    Text(
                        text = hour.label,
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(end = 6.dp)
                    )
                }
            }
        }

        // Single day column with absolute-positioned task blocks
        Box(
            modifier = Modifier
                .weight(1f)
                .height(gridHeightDp.dp)
                .border(1.dp, borderColor.copy(alpha = 0.5f))
        ) {
            // Hour cells (visual grid background)
            Column(modifier = Modifier.fillMaxSize()) {
                HOURS.forEach { _ ->
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(HOUR_HEIGHT_DP.dp)
                            .border(1.dp, borderColor.copy(alpha = 0.3f))
                    )
                }
            }

            // Task blocks (absolute-positioned over the hour cells)
            tasks.forEach { task ->
                val startMinutes = parseTimeToMinutes(task.start_time) - GRID_START_MINUTES
                val endMinutes = parseTimeToMinutes(task.end_time) - GRID_START_MINUTES
                val top = startMinutes.coerceAtLeast(0)
                val rawHeight = if (endMinutes > startMinutes) endMinutes - startMinutes else HOUR_HEIGHT_DP
                val height = rawHeight.coerceIn(HOUR_HEIGHT_DP / 2, gridHeightDp - top)
                val color = parseHexColor(task.color)
                val textColor = if (isLightColor(task.color)) Color(0xFF1F2937) else Color.White
                val isCompleted = task.recurrence_days?.isNotEmpty() == true && completions.contains(dateStr)

                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .offset(y = top.dp)
                        .height(height.dp)
                        .clip(RoundedCornerShape(6.dp))
                        .background(color.copy(alpha = if (isCompleted) 0.7f else 0.85f))
                        .clickable { onOpenTask(task.id) }
                        .padding(horizontal = 6.dp, vertical = 4.dp)
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        if (isCompleted) {
                            Text("✓", color = textColor, style = MaterialTheme.typography.labelSmall)
                        }
                        Column {
                            Text(
                                text = task.title,
                                style = MaterialTheme.typography.labelMedium,
                                color = textColor,
                                fontWeight = FontWeight.SemiBold,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                                textDecoration = if (isCompleted) TextDecoration.LineThrough else TextDecoration.None
                            )
                            Text(
                                text = timeRangeLabel(task),
                                style = MaterialTheme.typography.labelSmall,
                                color = textColor.copy(alpha = 0.85f),
                                maxLines = 1
                            )
                        }
                    }
                }
            }

            if (tasks.isEmpty()) {
                Text(
                    text = Localization.text("mobile.planner.noScheduled"),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier
                        .align(Alignment.TopCenter)
                        .padding(top = 24.dp)
                )
            }
        }
    }
}

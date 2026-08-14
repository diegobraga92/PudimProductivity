package com.pudimproductivity.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.taskService
import com.pudimproductivity.utils.formatWeekRange
import com.pudimproductivity.utils.getWeekDates
import kotlinx.coroutines.launch
import java.time.LocalDate

private val DAY_NAMES = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val DAY_LABELS = listOf("Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun")
private val DEFAULT_COLOR = Color(0xFF3B82F6)

/** Weekday key (mon..sun) for an ISO date. */
private fun weekdayName(isoDate: String): String {
    val date = LocalDate.parse(isoDate)
    return DAY_NAMES[(date.dayOfWeek.value + 6) % 7] // 0 = Mon
}

/** Whether a task is scheduled on the given weekday (habits recur, one-offs are date-based). */
private fun appliesOn(task: Task, day: String): Boolean {
    val recurrence = task.recurrence_days
    if (!recurrence.isNullOrEmpty()) return day in recurrence
    val scheduledDate = task.scheduled_date ?: return false
    return weekdayName(scheduledDate) == day
}

private fun parseHexColor(hex: String?): Color {
    if (hex == null || hex.length < 7) return DEFAULT_COLOR
    return try {
        Color(("FF" + hex.removePrefix("#")).toLong(16))
    } catch (_: Exception) {
        DEFAULT_COLOR
    }
}

private fun timeRangeLabel(task: Task): String {
    val start = task.start_time?.take(5) ?: "--:--"
    val end = task.end_time?.take(5) ?: "--:--"
    return "$start – $end"
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlannerScreen(onOpenTask: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    var weekOffset by remember { mutableStateOf(0) }
    var tasks by remember { mutableStateOf<List<Task>>(emptyList()) }
    var error by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    fun reload() {
        scope.launch {
            loading = true
            error = null
            try {
                tasks = ApiClient.taskService.listScheduledTasks()
            } catch (e: Exception) {
                error = e.message ?: "Failed to load planner"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    val weekDates = getWeekDates(weekOffset)

    Scaffold(
        topBar = { TopAppBar(title = { Text("Planner") }) }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp)
        ) {
            // Week navigation
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth()
            ) {
                TextButton(onClick = { weekOffset -= 1 }) { Text("‹ Prev") }
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    modifier = Modifier.weight(1f)
                ) {
                    Text(
                        text = if (weekOffset == 0) "This Week" else "Week $weekOffset",
                        style = MaterialTheme.typography.titleSmall
                    )
                    Text(
                        text = formatWeekRange(weekDates),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                TextButton(onClick = { weekOffset += 1 }) { Text("Next ›") }
            }

            Spacer(modifier = Modifier.height(8.dp))

            when {
                loading -> Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator()
                }
                error != null -> Text(error!!, color = MaterialTheme.colorScheme.error)
                tasks.isEmpty() -> Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = "No scheduled tasks yet. Schedule tasks from the Planner on the web to see them here.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
                else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    itemsIndexed(DAY_NAMES) { index, day ->
                        val date = weekDates.getOrNull(index) ?: return@itemsIndexed
                        val dayTasks = tasks
                            .filter { appliesOn(it, day) }
                            .sortedBy { it.start_time ?: "99:99" }
                        DayCard(
                            label = DAY_LABELS[index],
                            date = date,
                            tasks = dayTasks,
                            onOpenTask = onOpenTask
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun DayCard(
    label: String,
    date: String,
    tasks: List<Task>,
    onOpenTask: (String) -> Unit
) {
    Card(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp)
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = label,
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f)
                )
                Text(
                    text = date,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Spacer(modifier = Modifier.height(6.dp))
            if (tasks.isEmpty()) {
                Text(
                    text = "No scheduled tasks",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            } else {
                tasks.forEach { task ->
                    TaskRow(task = task, onClick = { onOpenTask(task.id) })
                    Spacer(modifier = Modifier.height(6.dp))
                }
            }
        }
    }
}

@Composable
private fun TaskRow(task: Task, onClick: () -> Unit) {
    val color = parseHexColor(task.color)
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(8.dp))
            .background(color.copy(alpha = 0.16f))
            .clickable(onClick = onClick)
            .padding(horizontal = 10.dp, vertical = 8.dp)
    ) {
        Box(
            modifier = Modifier
                .width(4.dp)
                .height(28.dp)
                .background(color, RoundedCornerShape(2.dp))
        )
        Spacer(modifier = Modifier.width(10.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = task.title,
                style = MaterialTheme.typography.bodyLarge,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            Text(
                text = timeRangeLabel(task),
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
        if (!task.recurrence_days.isNullOrEmpty()) {
            Text(
                text = "habit",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}


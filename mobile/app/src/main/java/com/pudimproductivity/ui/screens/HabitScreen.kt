package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.taskService
import com.pudimproductivity.ui.components.ProgressBar
import com.pudimproductivity.ui.components.ProgressVariant
import com.pudimproductivity.ui.components.StreakBadge
import com.pudimproductivity.ui.components.WeekHeatmap
import com.pudimproductivity.utils.computeStreaks
import com.pudimproductivity.utils.getToday
import com.pudimproductivity.utils.getWeekDates
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.launch

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val DAY_LABELS = mapOf(
    "mon" to "M", "tue" to "T", "wed" to "W", "thu" to "T",
    "fri" to "F", "sat" to "S", "sun" to "S"
)

/**
 * Dedicated habit screen (Phase 4): Material 3 chips for recurrence days,
 * streak badge, inline week heatmap with tap-to-toggle, and weekly progress.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HabitScreen(onBack: () -> Unit, onOpenTask: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    var habits by remember { mutableStateOf<List<Task>>(emptyList()) }
    var completionsMap by remember { mutableStateOf<Map<String, List<String>>>(emptyMap()) }
    var isLoading by remember { mutableStateOf(true) }
    var weekOffset by remember { mutableStateOf(0) }

    fun loadData() {
        scope.launch {
            isLoading = true
            try {
                habits = ApiClient.taskService.listTasks("habit")
                val weekDates = getWeekDates(weekOffset)
                val all = ApiClient.taskService.getAllTaskCompletions(weekDates.first(), weekDates.last())
                val map = mutableMapOf<String, List<String>>()
                for (task in habits) {
                    map[task.id] = all.filter { it.task_id == task.id }.map { it.completed_date }
                }
                completionsMap = map
            } catch (_: Exception) {
                habits = emptyList()
                completionsMap = emptyMap()
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) { loadData() }
    LaunchedEffect(weekOffset) { loadData() }

    @OptIn(FlowPreview::class)
    LaunchedEffect(Unit) {
        SyncClient.events.debounce(300).collect { loadData() }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Habits") },
                navigationIcon = { TextButton(onClick = onBack) { Text("← Back") } }
            )
        }
    ) { padding ->
        Column(modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.fillMaxWidth()
            ) {
                TextButton(onClick = { weekOffset -= 1 }) { Text("‹ Prev") }
                Text(
                    text = "Week ${if (weekOffset == 0) "current" else if (weekOffset < 0) "−${-weekOffset}" else "+$weekOffset"}",
                    style = MaterialTheme.typography.labelMedium,
                    modifier = Modifier.weight(1f),
                    textAlign = androidx.compose.ui.text.style.TextAlign.Center
                )
                TextButton(onClick = { weekOffset += 1 }) { Text("Next ›") }
            }

            Spacer(modifier = Modifier.height(8.dp))

            if (isLoading) {
                Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            } else if (habits.isEmpty()) {
                Text(
                    text = "No habits yet. Create a task with recurrence days to start building a habit.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            } else {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    items(habits, key = { it.id }) { task ->
                        HabitCard(
                            task = task,
                            completions = completionsMap[task.id] ?: emptyList(),
                            weekOffset = weekOffset,
                            onToggleDay = { date, isCompleted ->
                                scope.launch {
                                    try {
                                        if (isCompleted) {
                                            ApiClient.taskService.uncompleteTask(task.id, date)
                                        } else {
                                            ApiClient.taskService.completeTask(task.id, date)
                                        }
                                        loadData()
                                    } catch (_: Exception) { }
                                }
                            },
                            onClick = { onOpenTask(task.id) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun HabitCard(
    task: Task,
    completions: List<String>,
    weekOffset: Int,
    onToggleDay: (String, Boolean) -> Unit,
    onClick: () -> Unit
) {
    val today = getToday()
    val weekDates = getWeekDates(weekOffset)
    val scheduled = (task.recurrence_days ?: emptyList())
        .mapNotNull { day -> weekDates.getOrNull(DAY_ORDER.indexOf(day)) }
    val done = scheduled.count { it in completions }
    val streak = computeStreaks(completions)

    Card(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
        )
    ) {
        Column(modifier = Modifier.fillMaxWidth().padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                Text(
                    text = task.title,
                    style = MaterialTheme.typography.bodyLarge,
                    modifier = Modifier.weight(1f)
                )
                StreakBadge(result = streak)
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Material 3 chips for recurrence days
            Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                (task.recurrence_days ?: emptyList()).forEach { day ->
                    val doneToday = day == DAY_ORDER[getTodayIndex()] && today in completions
                    AssistChip(
                        onClick = {},
                        enabled = false,
                        label = { Text("${DAY_LABELS[day] ?: day}${if (doneToday) " ✓" else ""}") }
                    )
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            WeekHeatmap(
                recurrenceDays = task.recurrence_days ?: emptyList(),
                completions = completions,
                onToggleDay = onToggleDay,
                weekOffset = weekOffset,
                onWeekOffsetChange = { }
            )

            Spacer(modifier = Modifier.height(4.dp))

            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                Box(modifier = Modifier.weight(1f)) {
                    ProgressBar(
                        value = if (scheduled.isNotEmpty()) (done * 100) / scheduled.size else 0,
                        variant = ProgressVariant.HABIT,
                        height = 6.dp
                    )
                }
                Text(
                    text = "$done/${scheduled.size}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

private fun getTodayIndex(): Int {
    val today = getToday()
    val date = java.time.LocalDate.parse(today)
    return (date.dayOfWeek.value + 6) % 7 // 0 = Monday
}


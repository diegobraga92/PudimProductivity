package com.pudimproductivity.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.TaskCompletion
import com.pudimproductivity.api.UpdateTaskRequest
import kotlinx.coroutines.launch
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.util.Locale

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val DAY_SHORT = mapOf(
    "mon" to "M", "tue" to "T", "wed" to "W", "thu" to "T",
    "fri" to "F", "sat" to "S", "sun" to "S"
)

private fun getWeekDates(): List<String> {
    val today = LocalDate.now()
    val monday = today.with(DayOfWeek.MONDAY)
    return (0..6).map { monday.plusDays(it.toLong()).format(DateTimeFormatter.ISO_LOCAL_DATE) }
}

private fun getDayName(dateStr: String): String {
    val date = LocalDate.parse(dateStr)
    val dayIndex = (date.dayOfWeek.value + 6) % 7 // 0=Mon
    return DAY_ORDER[dayIndex]
}

@Composable
fun TaskListScreen(
    onCreateTask: () -> Unit,
    onTaskClick: (String) -> Unit
) {
    val scope = rememberCoroutineScope()
    var tasks by remember { mutableStateOf<List<Task>>(emptyList()) }
    var completionsMap by remember { mutableStateOf<Map<String, List<String>>>(emptyMap()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    fun loadData() {
        scope.launch {
            isLoading = true
            error = null
            try {
                tasks = ApiClient.taskService.listTasks()

                // Load completions for habit tasks
                val habitTasks = tasks.filter { it.recurrence_days != null && it.recurrence_days.isNotEmpty() }
                val weekDates = getWeekDates()
                val from = weekDates.first()
                val to = weekDates.last()
                val map = mutableMapOf<String, List<String>>()
                for (task in habitTasks) {
                    try {
                        val completions = ApiClient.taskService.getTaskCompletions(task.id, from, to)
                        map[task.id] = completions.map { it.completed_date }
                    } catch (_: Exception) {
                        map[task.id] = emptyList()
                    }
                }
                completionsMap = map
            } catch (e: Exception) {
                error = e.message ?: "Failed to load tasks"
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadData()
    }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        // Header
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "Tasks",
                style = MaterialTheme.typography.headlineMedium
            )
            Button(onClick = onCreateTask) {
                Text("+ New Task")
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        // Content
        when {
            isLoading -> {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator()
                }
            }
            error != null -> {
                Text(
                    text = "Error: $error",
                    color = MaterialTheme.colorScheme.error
                )
                Spacer(modifier = Modifier.height(8.dp))
                Button(onClick = { loadData() }) {
                    Text("Retry")
                }
            }
            tasks.isEmpty() -> {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = "No tasks yet. Create one!",
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
            else -> {
                LazyColumn(
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    items(tasks, key = { it.id }) { task ->
                        val isHabit = task.recurrence_days != null && task.recurrence_days.isNotEmpty()
                        val taskCompletions = completionsMap[task.id] ?: emptyList()

                        TaskCard(
                            task = task,
                            isHabit = isHabit,
                            completions = taskCompletions,
                            onToggleDone = {
                                scope.launch {
                                    try {
                                        val newStatus = if (task.status == "done") "todo" else "done"
                                        ApiClient.taskService.updateTask(
                                            task.id,
                                            UpdateTaskRequest(status = newStatus)
                                        )
                                        loadData()
                                    } catch (_: Exception) { }
                                }
                            },
                            onToggleHabitDay = { date, completed ->
                                scope.launch {
                                    try {
                                        if (completed) {
                                            ApiClient.taskService.uncompleteTask(task.id)
                                        } else {
                                            ApiClient.taskService.completeTask(task.id)
                                        }
                                        loadData()
                                    } catch (_: Exception) { }
                                }
                            },
                            onClick = { onTaskClick(task.id) }
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun TaskCard(
    task: Task,
    isHabit: Boolean,
    completions: List<String>,
    onToggleDone: () -> Unit,
    onToggleHabitDay: (String, Boolean) -> Unit,
    onClick: () -> Unit
) {
    val completedSet = completions.toSet()
    val weekDates = getWeekDates()

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() },
        colors = CardDefaults.cardColors(
            containerColor = if (isHabit)
                MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
            else if (task.status == "done")
                MaterialTheme.colorScheme.surfaceVariant
            else
                MaterialTheme.colorScheme.surface
        )
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp)
        ) {
            Row(
                verticalAlignment = Alignment.CenterVertically
            ) {
                if (!isHabit) {
                    Checkbox(
                        checked = task.status == "done",
                        onCheckedChange = { onToggleDone() }
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                }
                Text(
                    text = task.title,
                    style = MaterialTheme.typography.bodyLarge,
                    textDecoration = if (task.status == "done")
                        TextDecoration.LineThrough
                    else
                        TextDecoration.None,
                    color = if (task.status == "done")
                        MaterialTheme.colorScheme.onSurfaceVariant
                    else
                        MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.weight(1f)
                )
            }

            // Weekly habit streak
            if (isHabit) {
                Spacer(modifier = Modifier.height(6.dp))
                Row(
                    horizontalArrangement = Arrangement.spacedBy(4.dp)
                ) {
                    weekDates.forEach { date ->
                        val dayName = getDayName(date)
                        val isScheduled = task.recurrence_days?.contains(dayName) == true
                        val isCompleted = date in completedSet
                        val isToday = date == LocalDate.now().format(DateTimeFormatter.ISO_LOCAL_DATE)

                        Surface(
                            modifier = Modifier.size(28.dp),
                            shape = MaterialTheme.shapes.small,
                            color = when {
                                isCompleted -> MaterialTheme.colorScheme.primary
                                isScheduled -> MaterialTheme.colorScheme.tertiaryContainer
                                else -> MaterialTheme.colorScheme.surfaceVariant
                            },
                            contentColor = when {
                                isCompleted -> MaterialTheme.colorScheme.onPrimary
                                isScheduled -> MaterialTheme.colorScheme.onTertiaryContainer
                                else -> MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
                            },
                            onClick = {
                                if (isScheduled || isCompleted) {
                                    onToggleHabitDay(date, isCompleted)
                                }
                            }
                        ) {
                            Box(contentAlignment = Alignment.Center) {
                                Text(
                                    text = DAY_SHORT[dayName] ?: "",
                                    fontSize = 11.sp
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

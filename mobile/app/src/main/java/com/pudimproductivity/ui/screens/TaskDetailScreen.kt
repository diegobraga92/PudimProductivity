package com.pudimproductivity.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
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

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val DAY_LABELS = mapOf(
    "mon" to "Monday", "tue" to "Tuesday", "wed" to "Wednesday",
    "thu" to "Thursday", "fri" to "Friday", "sat" to "Saturday", "sun" to "Sunday"
)
private val DAY_SHORT = mapOf(
    "mon" to "Mon", "tue" to "Tue", "wed" to "Wed",
    "thu" to "Thu", "fri" to "Fri", "sat" to "Sat", "sun" to "Sun"
)

private fun getWeekDates(): List<String> {
    val today = LocalDate.now()
    val monday = today.with(DayOfWeek.MONDAY)
    return (0..6).map { monday.plusDays(it.toLong()).format(DateTimeFormatter.ISO_LOCAL_DATE) }
}

private fun getDayName(dateStr: String): String {
    val date = LocalDate.parse(dateStr)
    val dayIndex = (date.dayOfWeek.value + 6) % 7
    return DAY_ORDER[dayIndex]
}

@Composable
fun TaskDetailScreen(
    taskId: String,
    onUpdated: () -> Unit,
    onDeleted: () -> Unit,
    onBack: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var task by remember { mutableStateOf<Task?>(null) }
    var completions by remember { mutableStateOf<List<TaskCompletion>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var isEditing by remember { mutableStateOf(false) }
    var editTitle by remember { mutableStateOf("") }

    fun loadTask() {
        scope.launch {
            isLoading = true
            error = null
            try {
                task = ApiClient.taskService.getTask(taskId)
                // Load completions if habit
                val t = task
                if (t != null && t.recurrence_days != null && t.recurrence_days.isNotEmpty()) {
                    val weekDates = getWeekDates()
                    completions = ApiClient.taskService.getTaskCompletions(
                        taskId, weekDates.first(), weekDates.last()
                    )
                }
            } catch (e: Exception) {
                error = e.message ?: "Failed to load task"
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadTask()
    }

    val isHabit = task?.recurrence_days != null && task?.recurrence_days?.isNotEmpty() == true
    val completedDates = completions.map { it.completed_date }.toSet()
    val weekDates = getWeekDates()

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
            Column(modifier = Modifier.padding(16.dp)) {
                Text(
                    text = "Error: $error",
                    color = MaterialTheme.colorScheme.error
                )
                Spacer(modifier = Modifier.height(8.dp))
                Button(onClick = { loadTask() }) {
                    Text("Retry")
                }
                OutlinedButton(onClick = onBack) {
                    Text("Back")
                }
            }
        }
        task != null && isEditing -> {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(
                    text = "Edit Task",
                    style = MaterialTheme.typography.headlineMedium
                )
                Spacer(modifier = Modifier.height(16.dp))

                OutlinedTextField(
                    value = editTitle,
                    onValueChange = { editTitle = it },
                    label = { Text("Title") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth()
                )

                Spacer(modifier = Modifier.height(16.dp))

                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(
                        onClick = {
                            if (editTitle.isBlank()) return@Button
                            scope.launch {
                                try {
                                    ApiClient.taskService.updateTask(
                                        taskId,
                                        UpdateTaskRequest(title = editTitle.trim())
                                    )
                                    isEditing = false
                                    loadTask()
                                    onUpdated()
                                } catch (_: Exception) { }
                            }
                        }
                    ) {
                        Text("Save")
                    }
                    OutlinedButton(onClick = { isEditing = false }) {
                        Text("Cancel")
                    }
                }
            }
        }
        task != null -> {
            val t = task!!
            Column(modifier = Modifier.padding(16.dp)) {
                // Top bar
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    OutlinedButton(onClick = onBack) {
                        Text("← Back")
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(
                            onClick = {
                                editTitle = t.title
                                isEditing = true
                            }
                        ) {
                            Text("Edit")
                        }
                        Button(
                            onClick = {
                                scope.launch {
                                    try {
                                        ApiClient.taskService.deleteTask(taskId)
                                        onDeleted()
                                    } catch (_: Exception) { }
                                }
                            },
                            colors = ButtonDefaults.buttonColors(
                                containerColor = MaterialTheme.colorScheme.error
                            )
                        ) {
                            Text("Delete")
                        }
                    }
                }

                Spacer(modifier = Modifier.height(24.dp))

                // Title
                Text(
                    text = t.title,
                    style = MaterialTheme.typography.headlineSmall,
                    textDecoration = if (t.status == "done")
                        TextDecoration.LineThrough
                    else
                        TextDecoration.None,
                    color = if (t.status == "done")
                        MaterialTheme.colorScheme.onSurfaceVariant
                    else
                        MaterialTheme.colorScheme.onSurface
                )

                // Habit badge
                if (isHabit) {
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = "Habit · ${t.recurrence_days?.joinToString(", ") { DAY_LABELS[it] ?: it }}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.primary
                    )
                }

                Spacer(modifier = Modifier.height(16.dp))

                // One-off toggle
                if (!isHabit) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.clickable {
                            scope.launch {
                                try {
                                    val newStatus = if (t.status == "done") "todo" else "done"
                                    ApiClient.taskService.updateTask(
                                        taskId,
                                        UpdateTaskRequest(status = newStatus)
                                    )
                                    loadTask()
                                    onUpdated()
                                } catch (_: Exception) { }
                            }
                        }
                    ) {
                        Checkbox(
                            checked = t.status == "done",
                            onCheckedChange = null // handled by Row clickable
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = if (t.status == "done") "Mark as todo" else "Mark as done"
                        )
                    }
                }

                // Habit weekly calendar
                if (isHabit) {
                    Text(
                        text = "This Week",
                        style = MaterialTheme.typography.titleSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))

                    weekDates.forEach { date ->
                        val dayName = getDayName(date)
                        val isScheduled = t.recurrence_days?.contains(dayName) == true
                        val isCompleted = date in completedDates
                        val isToday = date == LocalDate.now().format(DateTimeFormatter.ISO_LOCAL_DATE)

                        Surface(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 2.dp),
                            shape = MaterialTheme.shapes.small,
                            color = when {
                                isCompleted -> MaterialTheme.colorScheme.primaryContainer
                                isScheduled -> MaterialTheme.colorScheme.tertiaryContainer
                                else -> MaterialTheme.colorScheme.surfaceVariant
                            },
                            onClick = {
                                if (isScheduled || isCompleted) {
                                    scope.launch {
                                        try {
                                            if (isCompleted) {
                                                ApiClient.taskService.uncompleteTask(taskId, date)
                                            } else {
                                                ApiClient.taskService.completeTask(taskId, date)
                                            }
                                            loadTask()
                                        } catch (_: Exception) { }
                                    }
                                }
                            }
                        ) {
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(horizontal = 12.dp, vertical = 8.dp),
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Text(
                                        text = DAY_SHORT[dayName] ?: "",
                                        style = MaterialTheme.typography.bodyMedium,
                                        fontWeight = if (isToday) androidx.compose.ui.text.font.FontWeight.Bold else androidx.compose.ui.text.font.FontWeight.Normal
                                    )
                                    Spacer(modifier = Modifier.width(8.dp))
                                    Text(
                                        text = date.substring(5),
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                                Text(
                                    text = when {
                                        isCompleted -> "✓ Completed"
                                        isScheduled -> "○ Pending"
                                        else -> "—"
                                    },
                                    style = MaterialTheme.typography.bodySmall,
                                    color = when {
                                        isCompleted -> MaterialTheme.colorScheme.primary
                                        isScheduled -> MaterialTheme.colorScheme.onTertiaryContainer
                                        else -> MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.5f)
                                    }
                                )
                            }
                        }
                    }
                }

                Spacer(modifier = Modifier.height(16.dp))

                // Timestamps
                Text(
                    text = "Created: ${t.created_at}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                Text(
                    text = "Updated: ${t.updated_at}",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

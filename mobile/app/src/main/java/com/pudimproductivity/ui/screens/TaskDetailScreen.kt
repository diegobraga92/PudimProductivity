package com.pudimproductivity.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.TaskCompletion
import com.pudimproductivity.api.UpdateTaskRequest
import com.pudimproductivity.api.taskService
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.ui.components.ProgressBar
import com.pudimproductivity.ui.components.ProgressVariant
import com.pudimproductivity.ui.components.StreakBadge
import com.pudimproductivity.ui.components.WeekHeatmap
import com.pudimproductivity.utils.computeStreaks
import com.pudimproductivity.utils.getToday
import com.pudimproductivity.utils.getWeekDates
import kotlinx.coroutines.launch
import java.time.LocalDate

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")

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
    var weekOffset by remember { mutableStateOf(0) }

    fun loadTask() {
        scope.launch {
            isLoading = true
            error = null
            try {
                task = ApiClient.taskService.getTask(taskId)
                // Load completions if habit
                val t = task
                if (t != null && t.recurrence_days != null && t.recurrence_days.isNotEmpty()) {
                    val weekDates = getWeekDates(weekOffset)
                    completions = ApiClient.taskService.getTaskCompletions(
                        taskId, weekDates.first(), weekDates.last()
                    )
                }
            } catch (e: Exception) {
                error = e.message ?: Localization.text("mobile.task.load.failed")
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadTask()
    }

    // Reload when week changes
    LaunchedEffect(weekOffset) {
        if (task != null && task!!.recurrence_days?.isNotEmpty() == true) {
            loadTask()
        }
    }

    val isHabit = task?.recurrence_days != null && task?.recurrence_days?.isNotEmpty() == true
    val completionsList = completions.map { it.completed_date }
    val weekDates = getWeekDates(weekOffset)
    val streakResult = computeStreaks(completionsList)
    val today = getToday()

    // Compute weekly progress
    val scheduledDays = task?.recurrence_days ?: emptyList()
    val weekScheduledDates = weekDates.filter { date ->
        val dayName = getDayName(date)
        scheduledDays.contains(dayName) && date <= today
    }
    val weeklyDone = completionsList.count { date -> weekScheduledDates.contains(date) && date <= today }
    val weeklyTotal = weekScheduledDates.size
    val weeklyPct = if (weeklyTotal > 0) (weeklyDone * 100) / weeklyTotal else 0

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
                    text = Localization.text("common.error") + ": $error",
                    color = MaterialTheme.colorScheme.error
                )
                Spacer(modifier = Modifier.height(8.dp))
                Button(onClick = { loadTask() }) {
                    Text(Localization.text("common.retry"))
                }
                OutlinedButton(onClick = onBack) {
                    Text(Localization.text("common.back"))
                }
            }
        }
        task != null && isEditing -> {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(
                    text = Localization.text("mobile.task.editTitle"),
                    style = MaterialTheme.typography.headlineMedium
                )
                Spacer(modifier = Modifier.height(16.dp))

                OutlinedTextField(
                    value = editTitle,
                    onValueChange = { editTitle = it },
                    label = { Text(Localization.text("common.title")) },
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
                        Text(Localization.text("common.save"))
                    }
                    OutlinedButton(onClick = { isEditing = false }) {
                        Text(Localization.text("common.cancel"))
                    }
                }
            }
        }
        task != null -> {
            val t = task!!
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(16.dp)
            ) {
                // Top bar
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    OutlinedButton(onClick = onBack) {
                        Text("← " + Localization.text("common.back"))
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Button(
                            onClick = {
                                editTitle = t.title
                                isEditing = true
                            }
                        ) {
                            Text(Localization.text("common.edit"))
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
                            Text(Localization.text("common.delete"))
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

                Spacer(modifier = Modifier.height(8.dp))

                // Habit badge + streak
                if (isHabit) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Text(
                            text = Localization.text(
                                "mobile.task.detail.habit",
                                "days" to (t.recurrence_days?.joinToString(", ") { Localization.text("days.$it") } ?: "")
                            ),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary
                        )
                        StreakBadge(result = streakResult)
                    }
                } else {
                    Text(
                        text = if (t.status == "done") Localization.text("common.done") else Localization.text("common.notDone"),
                        style = MaterialTheme.typography.bodySmall,
                        color = if (t.status == "done")
                            MaterialTheme.colorScheme.primary
                        else
                            MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                Spacer(modifier = Modifier.height(16.dp))

                // One-off toggle
                if (!isHabit) {
                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
                        )
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
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
                                .padding(12.dp)
                        ) {
                            Checkbox(
                                checked = t.status == "done",
                                onCheckedChange = null
                            )
                            Spacer(modifier = Modifier.width(8.dp))
                            Text(
                                text = if (t.status == "done") Localization.text("tasks.markTodo") else Localization.text("tasks.markDone")
                            )
                        }
                    }
                }

                // Habit view
                if (isHabit) {
                    Card(
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.2f)
                        )
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            // Weekly progress
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = Localization.text("mobile.task.detail.weeklyProgress"),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                                Text(
                                    text = "$weeklyDone/$weeklyTotal",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (weeklyTotal > 0 && weeklyDone >= weeklyTotal)
                                        MaterialTheme.colorScheme.primary
                                    else
                                        MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                            Spacer(modifier = Modifier.height(4.dp))
                            ProgressBar(
                                value = weeklyPct,
                                variant = ProgressVariant.HABIT,
                                height = 8.dp,
                                modifier = Modifier.fillMaxWidth()
                            )

                            Spacer(modifier = Modifier.height(12.dp))

                            // WeekHeatmap
                            WeekHeatmap(
                                recurrenceDays = t.recurrence_days ?: emptyList(),
                                completions = completionsList,
                                onToggleDay = { date, isCompleted ->
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
                                },
                                weekOffset = weekOffset,
                                onWeekOffsetChange = { newOffset ->
                                    weekOffset = newOffset
                                }
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.weight(1f))

                // Timestamps
                Column {
                    Text(
                        text = Localization.text("mobile.task.detail.created", "date" to t.created_at),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Text(
                        text = Localization.text("mobile.task.detail.updated", "date" to t.updated_at),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }
    }
}
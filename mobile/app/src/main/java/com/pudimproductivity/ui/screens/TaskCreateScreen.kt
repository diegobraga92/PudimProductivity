package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateTaskRequest
import kotlinx.coroutines.launch

private val DAYS = listOf(
    "mon" to "Mon",
    "tue" to "Tue",
    "wed" to "Wed",
    "thu" to "Thu",
    "fri" to "Fri",
    "sat" to "Sat",
    "sun" to "Sun"
)

@Composable
fun TaskCreateScreen(
    onCreated: () -> Unit,
    onCancel: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var title by remember { mutableStateOf("") }
    var isHabit by remember { mutableStateOf(false) }
    var selectedDays by remember { mutableStateOf(setOf<String>()) }
    var error by remember { mutableStateOf<String?>(null) }
    var isSubmitting by remember { mutableStateOf(false) }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp)
    ) {
        Text(
            text = "New Task",
            style = MaterialTheme.typography.headlineMedium
        )

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = title,
            onValueChange = { title = it },
            label = { Text("What do you need to do?") },
            placeholder = { Text("e.g. Have hair cut") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(12.dp))

        // Habit toggle
        Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
            Checkbox(
                checked = isHabit,
                onCheckedChange = {
                    isHabit = it
                    if (!it) selectedDays = emptySet()
                }
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text("Make this a habit (repeats weekly)")
        }

        // Day picker
        if (isHabit) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "Repeat on:",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(4.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                DAYS.forEach { (value, label) ->
                    val isSelected = value in selectedDays
                    FilterChip(
                        selected = isSelected,
                        onClick = {
                            selectedDays = if (isSelected) {
                                selectedDays - value
                            } else {
                                selectedDays + value
                            }
                        },
                        label = { Text(label) }
                    )
                }
            }
        }

        if (error != null) {
            Text(
                text = error!!,
                color = MaterialTheme.colorScheme.error,
                modifier = Modifier.padding(top = 8.dp)
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(
                onClick = {
                    if (title.isBlank()) {
                        error = "Title is required"
                        return@Button
                    }
                    if (isHabit && selectedDays.isEmpty()) {
                        error = "Select at least one day for the habit"
                        return@Button
                    }
                    isSubmitting = true
                    error = null
                    scope.launch {
                        try {
                            ApiClient.taskService.createTask(
                                CreateTaskRequest(
                                    title = title.trim(),
                                    recurrence_days = if (isHabit) selectedDays.toList() else null
                                )
                            )
                            onCreated()
                        } catch (e: Exception) {
                            error = e.message ?: "Failed to create task"
                        } finally {
                            isSubmitting = false
                        }
                    }
                },
                enabled = !isSubmitting
            ) {
                Text(if (isSubmitting) "Adding..." else "Add Task")
            }

            OutlinedButton(onClick = onCancel) {
                Text("Cancel")
            }
        }
    }
}

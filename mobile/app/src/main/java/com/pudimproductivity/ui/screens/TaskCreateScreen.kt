package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateTaskRequest
import com.pudimproductivity.api.ParseTaskRequest
import com.pudimproductivity.api.TaskList
import com.pudimproductivity.api.taskService
import com.pudimproductivity.i18n.Localization
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskCreateScreen(
    onCreated: () -> Unit,
    onCancel: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var title by remember { mutableStateOf("") }
    var isHabit by remember { mutableStateOf(false) }
    var selectedDays by remember { mutableStateOf(setOf<String>()) }
    var selectedListId by remember { mutableStateOf<String?>(null) }
    var taskLists by remember { mutableStateOf<List<TaskList>>(emptyList()) }
    var listDropdownExpanded by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var isSubmitting by remember { mutableStateOf(false) }

    // Load task lists for assignment
    LaunchedEffect(Unit) {
        try {
            taskLists = ApiClient.taskService.listTaskLists()
        } catch (_: Exception) { }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp)
    ) {
        Text(
            text = Localization.text("mobile.task.newTitle"),
            style = MaterialTheme.typography.headlineMedium
        )

        Spacer(modifier = Modifier.height(16.dp))

        // Smart Parse (Phase 7): quick natural-language entry with NLP preview.
        var parseInput by remember { mutableStateOf("") }
        var parseHint by remember { mutableStateOf<String?>(null) }
        TextButton(onClick = {
            scope.launch {
                try {
                    val result = ApiClient.taskService.parseTask(ParseTaskRequest(parseInput.trim()))
                    result.title?.let { title = it }
                    result.recurrence_days?.takeIf { it.isNotEmpty() }?.let {
                        isHabit = true
                        selectedDays = it.toSet()
                    }
                    parseHint = buildString {
                        result.title?.let { append(Localization.text("mobile.task.parseTitle", "value" to it) + "\n") }
                        result.due_date?.let { append(Localization.text("mobile.task.parseDue", "value" to it) + "\n") }
                        result.start_time?.let { append(Localization.text("mobile.task.parseAt", "value" to it)) }
                    }.trim().ifEmpty { null }
                } catch (e: Exception) {
                    parseHint = e.message ?: Localization.text("mobile.parse.failed")
                }
            }
        }, enabled = parseInput.isNotBlank()) { Text(Localization.text("mobile.task.smartAdd")) }
        OutlinedTextField(
            value = parseInput,
            onValueChange = { parseInput = it },
            label = { Text(Localization.text("mobile.task.parseLabel")) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth()
        )
        parseHint?.let { Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary) }

        Spacer(modifier = Modifier.height(12.dp))

        OutlinedTextField(
            value = title,
            onValueChange = { title = it },
            label = { Text(Localization.text("tasks.whatToDo")) },
            placeholder = { Text(Localization.text("tasks.titlePlaceholder")) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(12.dp))

        // Habit toggle
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(
                checked = isHabit,
                onCheckedChange = {
                    isHabit = it
                    if (!it) selectedDays = emptySet()
                }
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(Localization.text("tasks.makeHabit"))
        }

        // Day picker
        if (isHabit) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = Localization.text("tasks.repeatOn"),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(4.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                DAYS.forEach { (value, _) ->
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
                        label = { Text(Localization.text("days.$value")) }
                    )
                }
            }
        }

        // List assignment dropdown
        if (taskLists.isNotEmpty()) {
            Spacer(modifier = Modifier.height(12.dp))
            Text(
                text = Localization.text("mobile.task.listOptional"),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Spacer(modifier = Modifier.height(4.dp))

            ExposedDropdownMenuBox(
                expanded = listDropdownExpanded,
                onExpandedChange = { listDropdownExpanded = it }
            ) {
                OutlinedTextField(
                    value = taskLists.find { it.id == selectedListId }?.name ?: Localization.text("common.none"),
                    onValueChange = {},
                    readOnly = true,
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = listDropdownExpanded) },
                    modifier = Modifier
                        .fillMaxWidth()
                        .menuAnchor()
                )
                ExposedDropdownMenu(
                    expanded = listDropdownExpanded,
                    onDismissRequest = { listDropdownExpanded = false }
                ) {
                    // Option to clear selection
                    DropdownMenuItem(
                        text = { Text(Localization.text("common.none")) },
                        onClick = {
                            selectedListId = null
                            listDropdownExpanded = false
                        }
                    )
                    taskLists.forEach { list ->
                        DropdownMenuItem(
                            text = { Text(list.name) },
                            onClick = {
                                selectedListId = list.id
                                listDropdownExpanded = false
                            }
                        )
                    }
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
                        error = Localization.text("tasks.validation.titleRequired")
                        return@Button
                    }
                    if (isHabit && selectedDays.isEmpty()) {
                        error = Localization.text("tasks.validation.dayRequired")
                        return@Button
                    }
                    isSubmitting = true
                    error = null
                    scope.launch {
                        try {
                            ApiClient.taskService.createTask(
                                CreateTaskRequest(
                                    title = title.trim(),
                                    recurrence_days = if (isHabit) selectedDays.toList() else null,
                                    list_id = selectedListId
                                )
                            )
                            onCreated()
                        } catch (e: Exception) {
                            error = e.message ?: Localization.text("mobile.task.create.failed")
                        } finally {
                            isSubmitting = false
                        }
                    }
                },
                enabled = !isSubmitting
            ) {
                Text(if (isSubmitting) Localization.text("tasks.adding") else Localization.text("mobile.task.addTask"))
            }

            OutlinedButton(onClick = onCancel) {
                Text(Localization.text("common.cancel"))
            }
        }
    }
}
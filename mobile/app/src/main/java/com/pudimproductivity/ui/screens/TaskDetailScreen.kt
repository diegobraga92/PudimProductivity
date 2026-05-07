package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.UpdateTaskRequest
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskDetailScreen(
    taskId: String,
    onUpdated: () -> Unit,
    onDeleted: () -> Unit,
    onBack: () -> Unit
) {
    var task by remember { mutableStateOf<Task?>(null) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var isEditing by remember { mutableStateOf(false) }

    // Edit state
    var editTitle by remember { mutableStateOf("") }
    var editDescription by remember { mutableStateOf("") }
    var editStatus by remember { mutableStateOf("todo") }
    var editPriority by remember { mutableStateOf("medium") }
    var editDueDate by remember { mutableStateOf("") }

    val scope = rememberCoroutineScope()

    LaunchedEffect(taskId) {
        isLoading = true
        try {
            task = ApiClient.taskService.getTask(taskId)
        } catch (e: Exception) {
            error = e.message ?: "Failed to load task"
        } finally {
            isLoading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Task Details") },
                navigationIcon = {
                    TextButton(onClick = onBack) {
                        Text("Back")
                    }
                },
                actions = {
                    if (task != null && !isEditing) {
                        TextButton(onClick = {
                            task?.let {
                                editTitle = it.title
                                editDescription = it.description ?: ""
                                editStatus = it.status
                                editPriority = it.priority
                                editDueDate = it.due_date?.take(16) ?: ""
                                isEditing = true
                            }
                        }) {
                            Text("Edit")
                        }
                        TextButton(onClick = {
                            scope.launch {
                                try {
                                    ApiClient.taskService.deleteTask(taskId)
                                    onDeleted()
                                } catch (e: Exception) {
                                    error = e.message ?: "Failed to delete task"
                                }
                            }
                        }) {
                            Text("Delete", color = MaterialTheme.colorScheme.error)
                        }
                    }
                }
            )
        }
    ) { padding ->
        when {
            isLoading -> {
                Box(
                    modifier = Modifier.fillMaxSize().padding(padding),
                    contentAlignment = Alignment.Center
                ) {
                    CircularProgressIndicator()
                }
            }
            error != null -> {
                Box(
                    modifier = Modifier.fillMaxSize().padding(padding),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text("Error: $error", color = MaterialTheme.colorScheme.error)
                        Spacer(modifier = Modifier.height(8.dp))
                        Button(onClick = onBack) { Text("Back") }
                    }
                }
            }
            task == null -> {
                Box(
                    modifier = Modifier.fillMaxSize().padding(padding),
                    contentAlignment = Alignment.Center
                ) {
                    Text("Task not found")
                }
            }
            isEditing -> {
                EditTaskContent(
                    task = task!!,
                    editTitle = editTitle,
                    editDescription = editDescription,
                    editStatus = editStatus,
                    editPriority = editPriority,
                    editDueDate = editDueDate,
                    onTitleChange = { editTitle = it },
                    onDescriptionChange = { editDescription = it },
                    onStatusChange = { editStatus = it },
                    onPriorityChange = { editPriority = it },
                    onDueDateChange = { editDueDate = it },
                    onSave = {
                        scope.launch {
                            try {
                                ApiClient.taskService.updateTask(
                                    taskId,
                                    UpdateTaskRequest(
                                        title = editTitle.trim(),
                                        description = editDescription.trim().ifEmpty { null },
                                        status = editStatus,
                                        priority = editPriority,
                                        due_date = editDueDate.ifEmpty { null }
                                    )
                                )
                                task = ApiClient.taskService.getTask(taskId)
                                isEditing = false
                                onUpdated()
                            } catch (e: Exception) {
                                error = e.message ?: "Failed to update task"
                            }
                        }
                    },
                    onCancel = { isEditing = false },
                    modifier = Modifier.padding(padding)
                )
            }
            else -> {
                TaskDetailContent(
                    task = task!!,
                    modifier = Modifier.padding(padding)
                )
            }
        }
    }
}

@Composable
private fun TaskDetailContent(
    task: Task,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(16.dp)
            .verticalScroll(rememberScrollState())
    ) {
        Text(
            text = task.title,
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold
        )

        if (task.description != null) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = task.description,
                color = Color.DarkGray,
                lineHeight = 22.sp
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Status
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("Status: ", fontWeight = FontWeight.Bold)
            val statusColor = when (task.status) {
                "done" -> Color(0xFF2e7d32)
                "in_progress" -> Color(0xFFef6c00)
                else -> Color(0xFF1565c0)
            }
            val statusBg = when (task.status) {
                "done" -> Color(0xFFe8f5e9)
                "in_progress" -> Color(0xFFfff3e0)
                else -> Color(0xFFe3f2fd)
            }
            Surface(color = statusBg, shape = MaterialTheme.shapes.extraSmall) {
                Text(
                    text = task.status.replace("_", " "),
                    color = statusColor,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        // Priority
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text("Priority: ", fontWeight = FontWeight.Bold)
            val priorityColor = when (task.priority) {
                "high" -> Color(0xFFc62828)
                "medium" -> Color(0xFFef6c00)
                else -> Color(0xFF2e7d32)
            }
            val priorityBg = when (task.priority) {
                "high" -> Color(0xFFffebee)
                "medium" -> Color(0xFFfff3e0)
                else -> Color(0xFFe8f5e9)
            }
            Surface(color = priorityBg, shape = MaterialTheme.shapes.extraSmall) {
                Text(
                    text = task.priority.uppercase(),
                    color = priorityColor,
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                )
            }
        }

        // Due date
        if (task.due_date != null) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "Due: ${task.due_date}",
                fontWeight = FontWeight.Bold
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Timestamps
        Text(
            text = "Created: ${task.created_at}",
            fontSize = 12.sp,
            color = Color.Gray
        )
        Text(
            text = "Updated: ${task.updated_at}",
            fontSize = 12.sp,
            color = Color.Gray
        )
    }
}

@Composable
private fun EditTaskContent(
    task: Task,
    editTitle: String,
    editDescription: String,
    editStatus: String,
    editPriority: String,
    editDueDate: String,
    onTitleChange: (String) -> Unit,
    onDescriptionChange: (String) -> Unit,
    onStatusChange: (String) -> Unit,
    onPriorityChange: (String) -> Unit,
    onDueDateChange: (String) -> Unit,
    onSave: () -> Unit,
    onCancel: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier
            .fillMaxSize()
            .padding(16.dp)
            .verticalScroll(rememberScrollState())
    ) {
        Text(
            text = "Edit Task",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold
        )

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = editTitle,
            onValueChange = onTitleChange,
            label = { Text("Title *") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true
        )

        Spacer(modifier = Modifier.height(12.dp))

        OutlinedTextField(
            value = editDescription,
            onValueChange = onDescriptionChange,
            label = { Text("Description") },
            modifier = Modifier.fillMaxWidth(),
            minLines = 3,
            maxLines = 5
        )

        Spacer(modifier = Modifier.height(12.dp))

        // Status
        Text("Status", style = MaterialTheme.typography.labelLarge)
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            listOf("todo", "in_progress", "done").forEach { s ->
                FilterChip(
                    selected = editStatus == s,
                    onClick = { onStatusChange(s) },
                    label = { Text(s.replace("_", " ").replaceFirstChar { it.uppercase() }) }
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // Priority
        Text("Priority", style = MaterialTheme.typography.labelLarge)
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            listOf("low", "medium", "high").forEach { p ->
                FilterChip(
                    selected = editPriority == p,
                    onClick = { onPriorityChange(p) },
                    label = { Text(p.replaceFirstChar { it.uppercase() }) }
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        OutlinedTextField(
            value = editDueDate,
            onValueChange = onDueDateChange,
            label = { Text("Due Date (YYYY-MM-DDTHH:MM)") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true
        )

        Spacer(modifier = Modifier.height(16.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Button(
                onClick = onSave,
                modifier = Modifier.weight(1f)
            ) {
                Text("Save")
            }
            OutlinedButton(
                onClick = onCancel,
                modifier = Modifier.weight(1f)
            ) {
                Text("Cancel")
            }
        }
    }
}

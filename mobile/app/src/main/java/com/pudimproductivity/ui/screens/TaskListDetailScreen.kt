package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateTaskRequest
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.TaskList
import com.pudimproductivity.api.UpdateTaskListRequest
import com.pudimproductivity.api.UpdateTaskRequest
import com.pudimproductivity.api.taskService
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskListDetailScreen(
    listId: String,
    onBack: () -> Unit,
    onDeleted: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var taskList by remember { mutableStateOf<TaskList?>(null) }
    var tasks by remember { mutableStateOf<List<Task>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var newTitle by remember { mutableStateOf("") }
    var editingName by remember { mutableStateOf(false) }
    var editName by remember { mutableStateOf("") }

    fun loadData() {
        scope.launch {
            isLoading = true
            error = null
            try {
                taskList = ApiClient.taskService.getTaskList(listId)
                tasks = ApiClient.taskService.listTasksByListID(listId)
            } catch (e: Exception) {
                error = e.message ?: "Failed to load list"
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadData()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(taskList?.name ?: "Loading...") },
                navigationIcon = {
                    TextButton(onClick = onBack) {
                        Text("< Back")
                    }
                },
                actions = {
                    TextButton(onClick = {
                        if (taskList != null) {
                            editName = taskList!!.name
                            editingName = true
                        }
                    }) {
                        Text("Rename")
                    }
                    TextButton(onClick = {
                        scope.launch {
                            try {
                                ApiClient.taskService.deleteTaskList(listId)
                                onDeleted()
                            } catch (_: Exception) { }
                        }
                    }) {
                        Text("Delete", color = MaterialTheme.colorScheme.error)
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
            // Rename form
            if (editingName) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    OutlinedTextField(
                        value = editName,
                        onValueChange = { editName = it },
                        modifier = Modifier.weight(1f),
                        singleLine = true
                    )
                    Button(onClick = {
                        scope.launch {
                            try {
                                ApiClient.taskService.updateTaskList(listId, UpdateTaskListRequest(name = editName))
                                editingName = false
                                loadData()
                            } catch (_: Exception) { }
                        }
                    }) {
                        Text("Save")
                    }
                    TextButton(onClick = { editingName = false }) {
                        Text("Cancel")
                    }
                }
                Spacer(modifier = Modifier.height(8.dp))
            }

            // Quick-add form
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                OutlinedTextField(
                    value = newTitle,
                    onValueChange = { newTitle = it },
                    modifier = Modifier.weight(1f),
                    placeholder = { Text("Add a task...") },
                    singleLine = true
                )
                Button(
                    onClick = {
                        scope.launch {
                            try {
                                ApiClient.taskService.createTask(
                                    CreateTaskRequest(title = newTitle, list_id = listId)
                                )
                                newTitle = ""
                                loadData()
                            } catch (_: Exception) { }
                        }
                    },
                    enabled = newTitle.isNotBlank()
                ) {
                    Text("Add")
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

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
                    Text("Error: $error", color = MaterialTheme.colorScheme.error)
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
                            text = "No tasks in this list yet.",
                            style = MaterialTheme.typography.bodyLarge,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }
                }
                else -> {
                    LazyColumn(
                        verticalArrangement = Arrangement.spacedBy(4.dp)
                    ) {
                        items(tasks, key = { it.id }) { task ->
                            Card(
                                modifier = Modifier.fillMaxWidth(),
                                colors = CardDefaults.cardColors(
                                    containerColor = if (task.status == "done")
                                        MaterialTheme.colorScheme.surfaceVariant
                                    else
                                        MaterialTheme.colorScheme.surface
                                )
                            ) {
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(12.dp),
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Checkbox(
                                        checked = task.status == "done",
                                        onCheckedChange = {
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
                                        }
                                    )
                                    Spacer(modifier = Modifier.width(8.dp))
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
                            }
                        }
                    }
                }
            }
        }
    }
}

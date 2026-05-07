package com.pudimproductivity.ui.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Task
import kotlinx.coroutines.launch

@Composable
fun TaskListScreen(
    onCreateTask: () -> Unit,
    onTaskClick: (String) -> Unit
) {
    var tasks by remember { mutableStateOf<List<Task>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadTasks() {
        scope.launch {
            isLoading = true
            error = null
            try {
                tasks = ApiClient.taskService.listTasks()
            } catch (e: Exception) {
                error = e.message ?: "Failed to load tasks"
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadTasks()
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
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.Bold
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
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(vertical = 8.dp)
                )
                Button(onClick = { loadTasks() }) {
                    Text("Retry")
                }
            }
            tasks.isEmpty() -> {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = "No tasks found. Create one!",
                        color = Color.Gray
                    )
                }
            }
            else -> {
                LazyColumn(
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    items(tasks, key = { it.id }) { task ->
                        TaskCard(
                            task = task,
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
    onClick: () -> Unit
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(modifier = Modifier.padding(12.dp)) {
            Text(
                text = task.title,
                fontWeight = FontWeight.Bold,
                fontSize = 16.sp,
                textDecoration = if (task.status == "done") TextDecoration.LineThrough else TextDecoration.None
            )

            Spacer(modifier = Modifier.height(4.dp))

            Row(verticalAlignment = Alignment.CenterVertically) {
                // Priority badge
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

                Surface(
                    color = priorityBg,
                    shape = MaterialTheme.shapes.extraSmall
                ) {
                    Text(
                        text = task.priority.uppercase(),
                        color = priorityColor,
                        fontSize = 11.sp,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                    )
                }

                Spacer(modifier = Modifier.width(8.dp))

                // Status text
                Text(
                    text = task.status.replace("_", " "),
                    fontSize = 13.sp,
                    color = Color.Gray
                )

                // Due date
                if (task.due_date != null) {
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        text = "Due: ${task.due_date.take(10)}",
                        fontSize = 12.sp,
                        color = Color.Gray
                    )
                }
            }
        }
    }
}

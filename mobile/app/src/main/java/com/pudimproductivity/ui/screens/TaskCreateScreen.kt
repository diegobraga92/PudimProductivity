package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateTaskRequest
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskCreateScreen(
    onCreated: () -> Unit,
    onCancel: () -> Unit
) {
    var title by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var priority by remember { mutableStateOf("medium") }
    var dueDate by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    var isSubmitting by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Create Task") },
                navigationIcon = {
                    TextButton(onClick = onCancel) {
                        Text("Cancel")
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
                .verticalScroll(rememberScrollState())
        ) {
            // Title
            OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                label = { Text("Title *") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true
            )

            Spacer(modifier = Modifier.height(12.dp))

            // Description
            OutlinedTextField(
                value = description,
                onValueChange = { description = it },
                label = { Text("Description") },
                modifier = Modifier.fillMaxWidth(),
                minLines = 3,
                maxLines = 5
            )

            Spacer(modifier = Modifier.height(12.dp))

            // Priority
            Text("Priority", style = MaterialTheme.typography.labelLarge)
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                listOf("low", "medium", "high").forEach { p ->
                    FilterChip(
                        selected = priority == p,
                        onClick = { priority = p },
                        label = { Text(p.replaceFirstChar { it.uppercase() }) }
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // Due Date
            OutlinedTextField(
                value = dueDate,
                onValueChange = { dueDate = it },
                label = { Text("Due Date (YYYY-MM-DDTHH:MM)") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                placeholder = { Text("e.g. 2026-05-10T23:59") }
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Error
            if (error != null) {
                Text(
                    text = error!!,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(bottom = 8.dp)
                )
            }

            // Submit button
            Button(
                onClick = {
                    if (title.isBlank()) {
                        error = "Title is required"
                        return@Button
                    }
                    error = null
                    isSubmitting = true
                    scope.launch {
                        try {
                            ApiClient.taskService.createTask(
                                CreateTaskRequest(
                                    title = title.trim(),
                                    description = description.trim().ifEmpty { null },
                                    priority = priority,
                                    due_date = dueDate.ifEmpty { null }
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
                modifier = Modifier.fillMaxWidth(),
                enabled = !isSubmitting
            ) {
                Text(if (isSubmitting) "Creating..." else "Create Task")
            }
        }
    }
}

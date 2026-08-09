package com.pudimproductivity

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.HealthResponse
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.ui.screens.TaskCreateScreen
import com.pudimproductivity.ui.screens.TaskDetailScreen
import com.pudimproductivity.ui.screens.TaskListDetailScreen
import com.pudimproductivity.ui.screens.TaskListScreen
import com.pudimproductivity.ui.theme.PudimProductivityTheme
import kotlinx.coroutines.launch

enum class Screen {
    Health, TaskList, TaskCreate, TaskDetail, TaskListDetail
}

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Real-time task sync (Phase 2): app-lifetime WebSocket connection.
        SyncClient.start(applicationContext)

        setContent {
            PudimProductivityTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    AppNavigation()
                }
            }
        }
    }
}

@Composable
fun AppNavigation() {
    var currentScreen by remember { mutableStateOf(Screen.TaskList) }
    var selectedTaskId by remember { mutableStateOf<String?>(null) }
    var selectedListId by remember { mutableStateOf<String?>(null) }

    when (currentScreen) {
        Screen.Health -> {
            HealthScreen(
                onBack = { currentScreen = Screen.TaskList }
            )
        }
        Screen.TaskList -> {
            TaskListScreen(
                onCreateTask = { currentScreen = Screen.TaskCreate },
                onTaskClick = { taskId ->
                    selectedTaskId = taskId
                    currentScreen = Screen.TaskDetail
                },
                onListClick = { listId ->
                    selectedListId = listId
                    currentScreen = Screen.TaskListDetail
                }
            )
        }
        Screen.TaskCreate -> {
            TaskCreateScreen(
                onCreated = { currentScreen = Screen.TaskList },
                onCancel = { currentScreen = Screen.TaskList }
            )
        }
        Screen.TaskDetail -> {
            selectedTaskId?.let { taskId ->
                TaskDetailScreen(
                    taskId = taskId,
                    onUpdated = { currentScreen = Screen.TaskList },
                    onDeleted = { currentScreen = Screen.TaskList },
                    onBack = { currentScreen = Screen.TaskList }
                )
            }
        }
        Screen.TaskListDetail -> {
            selectedListId?.let { listId ->
                TaskListDetailScreen(
                    listId = listId,
                    onBack = { currentScreen = Screen.TaskList },
                    onDeleted = { currentScreen = Screen.TaskList }
                )
            }
        }
    }
}

@Composable
fun HealthScreen(onBack: () -> Unit) {
    var health by remember { mutableStateOf<HealthResponse?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                health = ApiClient.healthService.getHealth()
            } catch (e: Exception) {
                error = e.message ?: "Unknown error"
            }
        }
    }

    Column(modifier = Modifier.padding(24.dp)) {
        Text(
            text = "PudimProductivity",
            style = MaterialTheme.typography.headlineLarge
        )

        Text(
            text = "Backend Health",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = 16.dp)
        )

        when {
            health != null -> {
                val statusColor = when (health!!.status) {
                    "ok" -> MaterialTheme.colorScheme.primary
                    else -> MaterialTheme.colorScheme.error
                }
                val dbColor = when (health!!.db) {
                    "connected" -> MaterialTheme.colorScheme.primary
                    else -> MaterialTheme.colorScheme.error
                }

                Text(
                    text = "Status: ${health!!.status}",
                    color = statusColor,
                    modifier = Modifier.padding(top = 8.dp)
                )
                Text(text = "Version: ${health!!.version}")
                Text(
                    text = "Database: ${health!!.db}",
                    color = dbColor,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }
            error != null -> {
                Text(
                    text = "Error: $error",
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(top = 8.dp)
                )
            }
            else -> {
                Text(
                    text = "Checking backend status...",
                    modifier = Modifier.padding(top = 8.dp)
                )
            }
        }

        Spacer(modifier = Modifier.height(16.dp))
        Button(onClick = onBack) {
            Text("Back to Tasks")
        }
    }
}

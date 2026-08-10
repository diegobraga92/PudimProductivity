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
import com.pudimproductivity.data.TaskRepository
import com.pudimproductivity.fcm.ErrorReporter
import com.pudimproductivity.focus.FocusTimerManager
import com.pudimproductivity.notifications.HabitReminderScheduler
import com.pudimproductivity.sync.SyncScheduler
import com.pudimproductivity.ui.screens.BookListScreen
import com.pudimproductivity.ui.screens.DailyPlanScreen
import com.pudimproductivity.ui.screens.FocusTimerScreen
import com.pudimproductivity.ui.screens.HabitScreen
import com.pudimproductivity.ui.screens.InsightsScreen
import com.pudimproductivity.ui.screens.MealPlanScreen
import com.pudimproductivity.ui.screens.RecipeCreateScreen
import com.pudimproductivity.ui.screens.RecipeListScreen
import com.pudimproductivity.ui.screens.TaskCreateScreen
import com.pudimproductivity.ui.screens.TaskDetailScreen
import com.pudimproductivity.ui.screens.TaskListDetailScreen
import com.pudimproductivity.ui.screens.TaskListScreen
import com.pudimproductivity.ui.theme.PudimProductivityTheme
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

enum class Screen {
    Health, TaskList, TaskCreate, TaskDetail, TaskListDetail, FocusTimer, Habits, Recipes, RecipeCreate, Books, MealPlans, DailyPlan, Insights
}

class MainActivity : ComponentActivity() {

    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var repository: TaskRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Report uncaught exceptions to the backend error beacon.
        ErrorReporter.install(applicationContext)

        // Focus timer state + foreground service lifecycle.
        FocusTimerManager.init(applicationContext)

        // Real-time task sync (Phase 2): app-lifetime WebSocket connection.
        SyncClient.start(applicationContext)

        // Phase 9c offline-first: local repository + background sync workers.
        repository = TaskRepository(applicationContext, appScope)
        repository.start()
        SyncScheduler.schedule(applicationContext)
        HabitReminderScheduler.schedule(applicationContext)

        setContent {
            PudimProductivityTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    AppNavigation(repository)
                }
            }
        }
    }
}

@Composable
fun AppNavigation(repository: TaskRepository) {
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
                repository = repository,
                onCreateTask = { currentScreen = Screen.TaskCreate },
                onFocusTimer = { currentScreen = Screen.FocusTimer },
                onHabits = { currentScreen = Screen.Habits },
                onTaskClick = { taskId ->
                    selectedTaskId = taskId
                    currentScreen = Screen.TaskDetail
                },
                onListClick = { listId ->
                    selectedListId = listId
                    currentScreen = Screen.TaskListDetail
                },
                onRecipes = { currentScreen = Screen.Recipes },
                onBooks = { currentScreen = Screen.Books },
                onMealPlans = { currentScreen = Screen.MealPlans },
                onDailyPlan = { currentScreen = Screen.DailyPlan },
                onInsights = { currentScreen = Screen.Insights }
            )
        }
        Screen.FocusTimer -> {
            FocusTimerScreen(
                onBack = { currentScreen = Screen.TaskList }
            )
        }
        Screen.Habits -> {
            HabitScreen(
                onBack = { currentScreen = Screen.TaskList },
                onOpenTask = { taskId ->
                    selectedTaskId = taskId
                    currentScreen = Screen.TaskDetail
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
        Screen.Recipes -> {
            RecipeListScreen(
                onNew = { currentScreen = Screen.RecipeCreate },
                onBack = { currentScreen = Screen.TaskList }
            )
        }
        Screen.RecipeCreate -> {
            RecipeCreateScreen(onDone = { currentScreen = Screen.Recipes })
        }
        Screen.Books -> {
            BookListScreen(onBack = { currentScreen = Screen.TaskList })
        }
        Screen.MealPlans -> {
            MealPlanScreen(onBack = { currentScreen = Screen.TaskList })
        }
        Screen.DailyPlan -> {
            DailyPlanScreen()
        }
        Screen.Insights -> {
            InsightsScreen(onBack = { currentScreen = Screen.TaskList })
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

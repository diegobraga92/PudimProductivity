package com.pudimproductivity

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.EventNote
import androidx.compose.material.icons.filled.Checklist
import androidx.compose.material.icons.filled.HealthAndSafety
import androidx.compose.material.icons.filled.Insights
import androidx.compose.material.icons.filled.MoreHoriz
import androidx.compose.material.icons.filled.Movie
import androidx.compose.material.icons.filled.Repeat
import androidx.compose.material.icons.filled.Restaurant
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Timer
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.HealthResponse
import com.pudimproductivity.api.ServerConfig
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.data.TaskRepository
import com.pudimproductivity.fcm.ErrorReporter
import com.pudimproductivity.focus.FocusTimerManager
import com.pudimproductivity.notifications.HabitReminderScheduler
import com.pudimproductivity.sync.SyncScheduler
import com.pudimproductivity.ui.screens.FocusTimerScreen
import com.pudimproductivity.ui.screens.PlannerScreen
import com.pudimproductivity.ui.screens.HabitScreen
import com.pudimproductivity.ui.screens.InsightsScreen
import com.pudimproductivity.ui.screens.LibraryScreen
import com.pudimproductivity.ui.screens.RecipeCreateScreen
import com.pudimproductivity.ui.screens.RecipeListScreen
import com.pudimproductivity.ui.screens.TaskCreateScreen
import com.pudimproductivity.ui.screens.TaskDetailScreen
import com.pudimproductivity.ui.screens.TaskListDetailScreen
import com.pudimproductivity.ui.screens.ServerSettingsScreen
import com.pudimproductivity.ui.screens.TaskListScreen
import com.pudimproductivity.ui.theme.PudimProductivityTheme
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

enum class Screen {
    Health, ServerSettings, TaskList, TaskCreate, TaskDetail, TaskListDetail, FocusTimer, Habits, Recipes, RecipeCreate, Library, Planner, Insights
}

class MainActivity : ComponentActivity() {

    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private lateinit var repository: TaskRepository

    // Phase 10: deep link targets from home-screen widget taps (see
    // widget/TasksWidget.kt, widget/HabitsWidget.kt and [parseLaunch]).
    private val launchTarget = mutableStateOf<LaunchTarget?>(null)

    // Android 13+ notifications are a runtime permission. Habit reminders (and
    // FCM pushes) are silently dropped while it's denied, so ask up front; the
    // notification workers re-check before posting. The callback is a no-op.
    private val requestNotificationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Load the persisted backend URL before any API client is touched.
        ServerConfig.init(applicationContext)

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
        requestNotificationPermissionIfNeeded()

        // Widget taps arrive as extras on the launch intent (singleTop makes
        // onNewIntent deliver them when the activity is already resumed).
        launchTarget.value = parseLaunch(intent)

        setContent {
            PudimProductivityTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    AppNavigation(repository, launchTarget)
                }
            }
        }
    }

    /** Asks for POST_NOTIFICATIONS on Android 13+; no-op on older devices. */
    private fun requestNotificationPermissionIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) !=
            PackageManager.PERMISSION_GRANTED
        ) {
            requestNotificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        launchTarget.value = parseLaunch(intent)
    }

    private fun parseLaunch(intent: Intent?): LaunchTarget? {
        val screen = intent?.getStringExtra(EXTRA_SCREEN) ?: return null
        return when (screen) {
            // Task detail needs the id; without it the tap falls back to no-op.
            SCREEN_TASK_DETAIL -> intent.getStringExtra(EXTRA_TASK_ID)
                ?.let { LaunchTarget(Screen.TaskDetail, it) }
            SCREEN_TASKS -> LaunchTarget(Screen.TaskList, null)
            SCREEN_HABITS -> LaunchTarget(Screen.Habits, null)
            SCREEN_TASK_CREATE -> LaunchTarget(Screen.TaskCreate, null)
            else -> null
        }
    }

    companion object {
        // Widget deep-link extras — also used by the Glance widgets.
        const val EXTRA_SCREEN = "com.pudimproductivity.extra.SCREEN"
        const val EXTRA_TASK_ID = "com.pudimproductivity.extra.TASK_ID"

        const val SCREEN_TASKS = "tasks"
        const val SCREEN_TASK_DETAIL = "task_detail"
        const val SCREEN_HABITS = "habits"
        const val SCREEN_TASK_CREATE = "task_create"
    }
}

/** A widget deep-link target: which [Screen] to open, and with which task. */
private data class LaunchTarget(val screen: Screen, val taskId: String?)

/** Bottom-navigation top-level destinations. */
private enum class TopLevel { Tasks, Plan, Timer, Recipes, More }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun AppNavigation(
    repository: TaskRepository,
    launchTarget: MutableState<LaunchTarget?>
) {
    var currentScreen by remember { mutableStateOf(launchTarget.value?.screen ?: Screen.TaskList) }
    var selectedTaskId by remember { mutableStateOf(launchTarget.value?.taskId) }
    var selectedListId by remember { mutableStateOf<String?>(null) }
    var moreSheetOpen by remember { mutableStateOf(false) }

    // Phase 10: navigate to a widget tap target (also fires for the initial
    // launch), then clear it so the next tap is honoured.
    val target = launchTarget.value
    LaunchedEffect(target) {
        if (target != null) {
            currentScreen = target.screen
            selectedTaskId = target.taskId
            launchTarget.value = null
        }
    }

    // Map the current screen to the active bottom-nav item.
    val currentTopLevel = when (currentScreen) {
        Screen.TaskList, Screen.TaskCreate, Screen.TaskDetail,
        Screen.TaskListDetail, Screen.Habits -> TopLevel.Tasks
        Screen.Planner -> TopLevel.Plan
        Screen.FocusTimer -> TopLevel.Timer
        Screen.Recipes, Screen.RecipeCreate -> TopLevel.Recipes
        else -> TopLevel.More
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Column(modifier = Modifier.fillMaxSize()) {
            Box(modifier = Modifier.weight(1f)) {
                when (currentScreen) {
        Screen.Health -> {
            HealthScreen(
                onBack = { currentScreen = Screen.TaskList }
            )
        }
        Screen.ServerSettings -> {
            ServerSettingsScreen(
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
                onLibrary = { currentScreen = Screen.Library },
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
        Screen.Library -> {
            LibraryScreen(onBack = { currentScreen = Screen.TaskList })
        }
        Screen.Planner -> {
            PlannerScreen(
                onOpenTask = { taskId ->
                    selectedTaskId = taskId
                    currentScreen = Screen.TaskDetail
                }
            )
        }
        Screen.Insights -> {
            InsightsScreen(onBack = { currentScreen = Screen.TaskList })
        }
                }
            }

            // Persistent bottom navigation for the primary destinations.
            NavigationBar {
                NavigationBarItem(
                    selected = currentTopLevel == TopLevel.Tasks,
                    onClick = { currentScreen = Screen.TaskList },
                    icon = { Icon(Icons.Filled.Checklist, contentDescription = "Tasks") },
                    label = { Text("Tasks") }
                )
                NavigationBarItem(
                    selected = currentTopLevel == TopLevel.Plan,
                    onClick = { currentScreen = Screen.Planner },
                    icon = { Icon(Icons.AutoMirrored.Filled.EventNote, contentDescription = "Planner") },
                    label = { Text("Planner") }
                )
                NavigationBarItem(
                    selected = currentTopLevel == TopLevel.Timer,
                    onClick = { currentScreen = Screen.FocusTimer },
                    icon = { Icon(Icons.Filled.Timer, contentDescription = "Focus timer") },
                    label = { Text("Timer") }
                )
                NavigationBarItem(
                    selected = currentTopLevel == TopLevel.Recipes,
                    onClick = { currentScreen = Screen.Recipes },
                    icon = { Icon(Icons.Filled.Restaurant, contentDescription = "Recipes") },
                    label = { Text("Recipes") }
                )
                NavigationBarItem(
                    selected = currentTopLevel == TopLevel.More,
                    onClick = { moreSheetOpen = true },
                    icon = { Icon(Icons.Filled.MoreHoriz, contentDescription = "More") },
                    label = { Text("More") }
                )
            }
        }

        // "More" bottom sheet — secondary destinations.
        if (moreSheetOpen) {
            ModalBottomSheet(onDismissRequest = { moreSheetOpen = false }) {
                Column(modifier = Modifier.padding(bottom = 24.dp)) {
                    Text(
                        text = "More",
                        style = MaterialTheme.typography.titleLarge,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)
                    )
                    MoreSheetItem(Icons.Filled.Repeat, "Habits") {
                        moreSheetOpen = false
                        currentScreen = Screen.Habits
                    }
                    MoreSheetItem(Icons.Filled.Movie, "Library") {
                        moreSheetOpen = false
                        currentScreen = Screen.Library
                    }
                    MoreSheetItem(Icons.Filled.Insights, "Insights") {
                        moreSheetOpen = false
                        currentScreen = Screen.Insights
                    }
                    MoreSheetItem(Icons.Filled.HealthAndSafety, "Backend Health") {
                        moreSheetOpen = false
                        currentScreen = Screen.Health
                    }
                    MoreSheetItem(Icons.Filled.Settings, "Server Settings") {
                        moreSheetOpen = false
                        currentScreen = Screen.ServerSettings
                    }
                }
            }
        }
    }
}

@Composable
private fun MoreSheetItem(icon: ImageVector, label: String, onClick: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary
        )
        Spacer(modifier = Modifier.width(16.dp))
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge
        )
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

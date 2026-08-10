package com.pudimproductivity.ui.screens

import androidx.compose.foundation.clickable
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
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.TaskList
import com.pudimproductivity.api.UpdateTaskRequest
import com.pudimproductivity.api.taskService
import com.pudimproductivity.ui.components.ProgressBar
import com.pudimproductivity.ui.components.ProgressVariant
import com.pudimproductivity.ui.components.StreakBadge
import com.pudimproductivity.ui.components.WeekHeatmap
import com.pudimproductivity.utils.computeStreaks
import com.pudimproductivity.utils.getToday
import com.pudimproductivity.utils.getWeekDates
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.launch
import java.time.LocalDate

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")

private fun getDayName(dateStr: String): String {
    val date = LocalDate.parse(dateStr)
    val dayIndex = (date.dayOfWeek.value + 6) % 7 // 0=Mon
    return DAY_ORDER[dayIndex]
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskListScreen(
    onCreateTask: () -> Unit,
    onFocusTimer: () -> Unit,
    onHabits: () -> Unit,
    onTaskClick: (String) -> Unit,
    onListClick: (String) -> Unit,
    onRecipes: () -> Unit,
    onBooks: () -> Unit,
    onMealPlans: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var tasks by remember { mutableStateOf<List<Task>>(emptyList()) }
    var taskLists by remember { mutableStateOf<List<TaskList>>(emptyList()) }
    var completionsMap by remember { mutableStateOf<Map<String, List<String>>>(emptyMap()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedTab by remember { mutableStateOf(0) }
    var newTodoTitle by remember { mutableStateOf("") }
    var newHabitTitle by remember { mutableStateOf("") }
    var newListName by remember { mutableStateOf("") }

    // Week offset for habits (shared across all habits)
    var habitWeekOffset by remember { mutableStateOf(0) }

    fun loadData() {
        scope.launch {
            isLoading = true
            error = null
            try {
                val todoResult = ApiClient.taskService.listTasks("one-off")
                val habitResult = ApiClient.taskService.listTasks("habit")
                tasks = todoResult + habitResult
                taskLists = ApiClient.taskService.listTaskLists()

                // Load completions for all habits in one batch call
                val weekDates = getWeekDates(habitWeekOffset)
                val from = weekDates.first()
                val to = weekDates.last()
                val allCompletions = try {
                    ApiClient.taskService.getAllTaskCompletions(from, to)
                } catch (_: Exception) {
                    emptyList()
                }

                val map = mutableMapOf<String, List<String>>()
                for (task in habitResult) {
                    map[task.id] = allCompletions
                        .filter { it.task_id == task.id }
                        .map { it.completed_date }
                }
                completionsMap = map
            } catch (e: Exception) {
                error = e.message ?: "Failed to load tasks"
            } finally {
                isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadData()
    }

    // Reload data in response to real-time task events from any client.
    // Debounced to coalesce bursts of events into a single refresh.
    @OptIn(FlowPreview::class)
    LaunchedEffect(Unit) {
        SyncClient.events
            .debounce(300)
            .collect {
                loadData()
            }
    }

    // Reload completions when week changes
    LaunchedEffect(habitWeekOffset) {
        if (selectedTab == 1) {
            loadData()
        }
    }

    val todoTasks = tasks.filter { it.recurrence_days == null || it.recurrence_days.isEmpty() }
    val habitTasks = tasks.filter { it.recurrence_days != null && it.recurrence_days.isNotEmpty() }

    // Compute today's habit completions
    val today = getToday()
    val todayHabitCompletions = completionsMap.values.count { dates -> today in dates }
    val habitProgress = if (habitTasks.isNotEmpty()) {
        (todayHabitCompletions * 100) / habitTasks.size
    } else 0
    val doneTodos = todoTasks.count { it.status == "done" }
    val todoProgress = if (todoTasks.isNotEmpty()) {
        (doneTodos * 100) / todoTasks.size
    } else 0

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Tasks") },
                actions = {
                    Button(onClick = onHabits) {
                        Text("📊 Habits")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(onClick = onFocusTimer) {
                        Text("⏱ Focus")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(onClick = onRecipes) {
                        Text("🍳")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(onClick = onBooks) {
                        Text("📚")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(onClick = onMealPlans) {
                        Text("🗓")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(onClick = onCreateTask) {
                        Text("+ New Task")
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
            // Today's Progress Row
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(
                        text = "Today's Progress",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(16.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        // Todo progress
                        Column(modifier = Modifier.weight(1f)) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                Text(
                                    text = "$doneTodos/${todoTasks.size}",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.primary
                                )
                                ProgressBar(
                                    value = todoProgress,
                                    variant = ProgressVariant.TODO,
                                    height = 6.dp,
                                    modifier = Modifier.weight(1f)
                                )
                            }
                        }
                        // Habit progress
                        Column(modifier = Modifier.weight(1f)) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(8.dp)
                            ) {
                                Text(
                                    text = "$todayHabitCompletions/${habitTasks.size}",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.primary
                                )
                                ProgressBar(
                                    value = habitProgress,
                                    variant = ProgressVariant.HABIT,
                                    height = 6.dp,
                                    modifier = Modifier.weight(1f)
                                )
                            }
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Tabs
            TabRow(selectedTabIndex = selectedTab) {
                Tab(
                    selected = selectedTab == 0,
                    onClick = { selectedTab = 0 },
                    text = { Text("To-Do (${todoTasks.size})") }
                )
                Tab(
                    selected = selectedTab == 1,
                    onClick = { selectedTab = 1 },
                    text = { Text("Habits (${habitTasks.size})") }
                )
                Tab(
                    selected = selectedTab == 2,
                    onClick = { selectedTab = 2 },
                    text = { Text("Lists (${taskLists.size})") }
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

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
                selectedTab == 0 -> {
                    // TODO TAB
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        OutlinedTextField(
                            value = newTodoTitle,
                            onValueChange = { newTodoTitle = it },
                            modifier = Modifier.weight(1f),
                            placeholder = { Text("Quick add todo...") },
                            singleLine = true
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    try {
                                        ApiClient.taskService.createTask(CreateTaskRequest(title = newTodoTitle))
                                        newTodoTitle = ""
                                        loadData()
                                    } catch (_: Exception) { }
                                }
                            },
                            enabled = newTodoTitle.isNotBlank()
                        ) {
                            Text("Add")
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    if (todoTasks.isEmpty()) {
                        Box(
                            modifier = Modifier.fillMaxSize(),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                text = "No todos yet. Create one!",
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    } else {
                        LazyColumn(
                            verticalArrangement = Arrangement.spacedBy(4.dp)
                        ) {
                            items(todoTasks, key = { it.id }) { task ->
                                Card(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable { onTaskClick(task.id) },
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
                selectedTab == 1 -> {
                    // HABITS TAB
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        OutlinedTextField(
                            value = newHabitTitle,
                            onValueChange = { newHabitTitle = it },
                            modifier = Modifier.weight(1f),
                            placeholder = { Text("Quick add habit (weekdays)...") },
                            singleLine = true
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    try {
                                        ApiClient.taskService.createTask(
                                            CreateTaskRequest(
                                                title = newHabitTitle,
                                                recurrence_days = listOf("mon", "tue", "wed", "thu", "fri")
                                            )
                                        )
                                        newHabitTitle = ""
                                        loadData()
                                    } catch (_: Exception) { }
                                }
                            },
                            enabled = newHabitTitle.isNotBlank()
                        ) {
                            Text("Add")
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    if (habitTasks.isEmpty()) {
                        Box(
                            modifier = Modifier.fillMaxSize(),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                text = "No habits yet. Create one!",
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    } else {
                        LazyColumn(
                            verticalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            items(habitTasks, key = { it.id }) { task ->
                                val taskCompletions = completionsMap[task.id] ?: emptyList()
                                val weekDates = getWeekDates(habitWeekOffset)

                                // Compute stats
                                val todayStr = getToday()
                                val scheduledDays = task.recurrence_days ?: emptyList()
                                val weekScheduledDates = weekDates.filter { date ->
                                    val dayName = getDayName(date)
                                    scheduledDays.contains(dayName) && date <= todayStr
                                }
                                val weeklyDone = taskCompletions.count { date ->
                                    weekScheduledDates.contains(date) && date <= todayStr
                                }
                                val weeklyTotal = weekScheduledDates.size
                                val weeklyPct = if (weeklyTotal > 0) (weeklyDone * 100) / weeklyTotal else 0

                                val streakResult = computeStreaks(taskCompletions)

                                Card(
                                    modifier = Modifier.fillMaxWidth(),
                                    colors = CardDefaults.cardColors(
                                        containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
                                    )
                                ) {
                                    Column(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(12.dp)
                                    ) {
                                        // Title row with streak and delete
                                        Row(
                                            verticalAlignment = Alignment.CenterVertically,
                                            modifier = Modifier.fillMaxWidth()
                                        ) {
                                            Text(
                                                text = task.title,
                                                style = MaterialTheme.typography.bodyLarge,
                                                modifier = Modifier.weight(1f)
                                            )
                                            StreakBadge(result = streakResult)
                                            Spacer(modifier = Modifier.width(8.dp))
                                            TextButton(
                                                onClick = {
                                                    scope.launch {
                                                        try {
                                                            ApiClient.taskService.deleteTask(task.id)
                                                            loadData()
                                                        } catch (_: Exception) { }
                                                    }
                                                }
                                            ) {
                                                Text(
                                                    "✕",
                                                    color = MaterialTheme.colorScheme.error,
                                                    style = MaterialTheme.typography.labelSmall
                                                )
                                            }
                                        }

                                        Spacer(modifier = Modifier.height(6.dp))

                                        // WeekHeatmap with navigation
                                        WeekHeatmap(
                                            recurrenceDays = task.recurrence_days ?: emptyList(),
                                            completions = taskCompletions,
                                            onToggleDay = { date, isCompleted ->
                                                scope.launch {
                                                    try {
                                                        if (isCompleted) {
                                                            ApiClient.taskService.uncompleteTask(task.id, date)
                                                        } else {
                                                            ApiClient.taskService.completeTask(task.id, date)
                                                        }
                                                        loadData()
                                                    } catch (_: Exception) { }
                                                }
                                            },
                                            weekOffset = habitWeekOffset,
                                            onWeekOffsetChange = { newOffset ->
                                                habitWeekOffset = newOffset
                                            }
                                        )

                                        Spacer(modifier = Modifier.height(4.dp))

                                        // Compact progress bar
                                        Row(
                                            verticalAlignment = Alignment.CenterVertically,
                                            horizontalArrangement = Arrangement.spacedBy(8.dp),
                                            modifier = Modifier.fillMaxWidth()
                                        ) {
                                            Box(modifier = Modifier.weight(1f)) {
                                                ProgressBar(
                                                    value = weeklyPct,
                                                    variant = ProgressVariant.HABIT,
                                                    height = 6.dp
                                                )
                                            }
                                            Text(
                                                text = if (weeklyTotal > 0 && weeklyDone >= weeklyTotal) "✅ $weeklyDone/$weeklyTotal" else "$weeklyDone/$weeklyTotal",
                                                style = MaterialTheme.typography.labelSmall,
                                                color = if (weeklyTotal > 0 && weeklyDone >= weeklyTotal)
                                                    MaterialTheme.colorScheme.primary
                                                else
                                                    MaterialTheme.colorScheme.onSurfaceVariant
                                            )
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
                selectedTab == 2 -> {
                    // LISTS TAB
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        OutlinedTextField(
                            value = newListName,
                            onValueChange = { newListName = it },
                            modifier = Modifier.weight(1f),
                            placeholder = { Text("New list name...") },
                            singleLine = true
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    try {
                                        ApiClient.taskService.createTaskList(
                                            com.pudimproductivity.api.CreateTaskListRequest(name = newListName)
                                        )
                                        newListName = ""
                                        loadData()
                                    } catch (_: Exception) { }
                                }
                            },
                            enabled = newListName.isNotBlank()
                        ) {
                            Text("Create")
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    if (taskLists.isEmpty()) {
                        Box(
                            modifier = Modifier.fillMaxSize(),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                text = "No lists yet. Create one above!",
                                style = MaterialTheme.typography.bodyLarge,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                        }
                    } else {
                        LazyColumn(
                            verticalArrangement = Arrangement.spacedBy(4.dp)
                        ) {
                            items(taskLists, key = { it.id }) { list ->
                                Card(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable { onListClick(list.id) }
                                ) {
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(16.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Text(
                                            text = list.name,
                                            style = MaterialTheme.typography.titleMedium,
                                            modifier = Modifier.weight(1f)
                                        )
                                        Text(
                                            text = ">",
                                            color = MaterialTheme.colorScheme.onSurfaceVariant
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
}
package com.pudimproductivity.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.api.TaskList
import com.pudimproductivity.ui.components.ProgressBar
import com.pudimproductivity.ui.components.ProgressVariant
import com.pudimproductivity.ui.components.StreakBadge
import com.pudimproductivity.ui.components.WeekHeatmap
import com.pudimproductivity.utils.computeStreaks
import com.pudimproductivity.utils.getToday
import com.pudimproductivity.utils.getWeekDates
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.debounce
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
    repository: com.pudimproductivity.data.TaskRepository,
    onCreateTask: () -> Unit,
    onFocusTimer: () -> Unit,
    onHabits: () -> Unit,
    onTaskClick: (String) -> Unit,
    onListClick: (String) -> Unit,
    onRecipes: () -> Unit,
    onBooks: () -> Unit,
    onMealPlans: () -> Unit,
    onDailyPlan: () -> Unit,
    onInsights: () -> Unit
) {
    // Phase 9c: local-first — read straight from the local database via the
    // repository's flows (instant + offline); writes go through the repository
    // (optimistic + dirty flag) and are flushed to the server on sync.
    val tasks by repository.tasks.collectAsState()
    val taskLists by repository.taskLists.collectAsState()
    val completions by repository.completions.collectAsState()
    val isOnline by repository.online.collectAsState()

    var selectedTab by remember { mutableStateOf(0) }
    var newTodoTitle by remember { mutableStateOf("") }
    var newHabitTitle by remember { mutableStateOf("") }
    var newListName by remember { mutableStateOf("") }
    // Phase 8: list currently open in the share dialog.
    var shareList by remember { mutableStateOf<TaskList?>(null) }

    // Week offset for habits (shared across all habits)
    var habitWeekOffset by remember { mutableStateOf(0) }

    val completionsMap = remember(completions) {
        completions.groupBy({ it.task_id }, { it.completed_date })
    }

    // Phase 9c: WS events still arrive (Phase 2) — refresh local state so a
    // change made on another client is reflected immediately.
    @OptIn(FlowPreview::class)
    LaunchedEffect(Unit) {
        SyncClient.events
            .debounce(300)
            .collect {
                repository.refreshFromLocal()
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
                    Button(onClick = onDailyPlan) {
                        Text("🤖")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(onClick = onInsights) {
                        Text("🧠")
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
            PrimaryTabRow(selectedTabIndex = selectedTab) {
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

            // Phase 9c: offline banner — data is local-first, so the app stays
            // usable; this just informs the user that sync is pending.
            if (!isOnline) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    color = MaterialTheme.colorScheme.errorContainer,
                    shape = MaterialTheme.shapes.small
                ) {
                    Text(
                        text = "📡 Offline — changes will sync when you reconnect",
                        modifier = Modifier.padding(10.dp),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onErrorContainer
                    )
                }
                Spacer(modifier = Modifier.height(8.dp))
            }

            when (selectedTab) {
                0 -> {
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
                                repository.createTask(title = newTodoTitle)
                                newTodoTitle = ""
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
                                        // Left accent strip — brand identity, mirrors web cards
                                        Box(
                                            modifier = Modifier
                                                .width(3.dp)
                                                .height(20.dp)
                                                .clip(RoundedCornerShape(2.dp))
                                                .background(MaterialTheme.colorScheme.primary)
                                        )
                                        Spacer(modifier = Modifier.width(10.dp))
                                        Checkbox(
                                            checked = task.status == "done",
                                            onCheckedChange = {
                                                val newStatus = if (task.status == "done") "todo" else "done"
                                                repository.updateTask(task.id, status = newStatus)
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
                1 -> {
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
                                repository.createTask(
                                    title = newHabitTitle,
                                    recurrenceDays = listOf("mon", "tue", "wed", "thu", "fri")
                                )
                                newHabitTitle = ""
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
                                                    repository.deleteTask(task.id)
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
                                                if (isCompleted) {
                                                    repository.uncompleteHabit(task.id, date)
                                                } else {
                                                    repository.completeHabit(task.id, date)
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
                2 -> {
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
                                repository.createTaskList(newListName)
                                newListName = ""
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
                                        // Phase 8: share dialog.
                                        TextButton(onClick = { shareList = list }) {
                                            Text("👥")
                                        }
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

    // Phase 8: share dialog.
    shareList?.let { list ->
        TaskListShareDialog(taskList = list, onClose = { shareList = null })
    }
}
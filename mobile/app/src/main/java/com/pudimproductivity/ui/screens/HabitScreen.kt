package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.taskService
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.ui.components.SortSelector
import com.pudimproductivity.ui.components.StreakBadge
import com.pudimproductivity.ui.components.WeekHeatmap
import com.pudimproductivity.ui.components.WeekNavigator
import com.pudimproductivity.utils.SortOption
import com.pudimproductivity.utils.TaskSortPreferences
import com.pudimproductivity.utils.computeStreaks
import com.pudimproductivity.utils.getRollingWindowDates
import com.pudimproductivity.utils.sortTasks
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.launch

// Habit ordering (mirrors web's Habits column sort, including time options).
private const val HABIT_SORT_KEY = "taskSort.habits"

private val HABIT_SORT_OPTIONS = listOf(
    SortOption.CREATED_DESC, SortOption.CREATED_ASC,
    SortOption.ALPHA_ASC, SortOption.ALPHA_DESC,
    SortOption.TIME_ASC, SortOption.TIME_DESC
)

/**
 * Dedicated habit screen (Phase 4): Material 3 chips for recurrence days,
 * streak badge, inline week heatmap with tap-to-toggle, and weekly progress.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HabitScreen(onBack: () -> Unit, onOpenTask: (String) -> Unit) {
    val scope = rememberCoroutineScope()
    var habits by remember { mutableStateOf<List<Task>>(emptyList()) }
    var completionsMap by remember { mutableStateOf<Map<String, List<String>>>(emptyMap()) }
    var isLoading by remember { mutableStateOf(true) }
    var isRefreshing by remember { mutableStateOf(false) }
    var weekOffset by remember { mutableStateOf(0) }
    val context = LocalContext.current
    var habitSort by remember { mutableStateOf(TaskSortPreferences.load(context, HABIT_SORT_KEY)) }

    val sortedHabits = sortTasks(habits, habitSort)

    /** Fetches habits + completions for the visible rolling window (no loading flag). */
    suspend fun fetchData() {
        try {
            habits = ApiClient.taskService.listTasks("habit")
            val weekDates = getRollingWindowDates(weekOffset)
            val all = ApiClient.taskService.getAllTaskCompletions(weekDates.first(), weekDates.last())
            val map = mutableMapOf<String, List<String>>()
            for (task in habits) {
                map[task.id] = all.filter { it.task_id == task.id }.map { it.completed_date }
            }
            completionsMap = map
        } catch (_: Exception) {
            habits = emptyList()
            completionsMap = emptyMap()
        }
    }

    fun loadData() {
        scope.launch {
            isLoading = true
            fetchData()
            isLoading = false
        }
    }

    LaunchedEffect(Unit) { loadData() }
    LaunchedEffect(weekOffset) { loadData() }

    @OptIn(FlowPreview::class)
    LaunchedEffect(Unit) {
        SyncClient.events.debounce(300).collect { loadData() }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(Localization.text("tasks.habits")) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = Localization.text("common.back"))
                    }
                }
            )
        }
    ) { padding ->
        // Pull-to-refresh: re-fetches habits + completions from the server.
        PullToRefreshBox(
            isRefreshing = isRefreshing,
            onRefresh = {
                scope.launch {
                    isRefreshing = true
                    fetchData()
                    isRefreshing = false
                }
            },
            modifier = Modifier.fillMaxSize().padding(padding)
        ) {
            Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
                // Single week selector shared by all habit cards
                WeekNavigator(
                    weekOffset = weekOffset,
                    onWeekOffsetChange = { weekOffset = it }
                )

            Spacer(modifier = Modifier.height(4.dp))

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End
            ) {
                SortSelector(
                    value = habitSort,
                    options = HABIT_SORT_OPTIONS,
                    onSelect = { option ->
                        habitSort = option
                        TaskSortPreferences.save(context, HABIT_SORT_KEY, option)
                    }
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            if (isLoading) {
                Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            } else if (habits.isEmpty()) {
                Text(
                    text = Localization.text("mobile.habits.empty"),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            } else {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    items(sortedHabits, key = { it.id }) { task ->
                        HabitCard(
                            task = task,
                            completions = completionsMap[task.id] ?: emptyList(),
                            weekOffset = weekOffset,
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
                            onClick = { onOpenTask(task.id) }
                        )
                    }
                }
            }
        }
        }
    }
}

@Composable
private fun HabitCard(
    task: Task,
    completions: List<String>,
    weekOffset: Int,
    onToggleDay: (String, Boolean) -> Unit,
    onClick: () -> Unit
) {
    val streak = computeStreaks(completions)

    Card(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.large,
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface
        )
    ) {
        Column(modifier = Modifier.fillMaxWidth().padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.fillMaxWidth()) {
                Text(
                    text = task.title,
                    style = MaterialTheme.typography.bodyLarge,
                    modifier = Modifier.weight(1f)
                )
                StreakBadge(result = streak)
            }

            Spacer(modifier = Modifier.height(8.dp))

            WeekHeatmap(
                recurrenceDays = task.recurrence_days ?: emptyList(),
                completions = completions,
                onToggleDay = onToggleDay,
                weekOffset = weekOffset,
                onWeekOffsetChange = null
            )
        }
    }
}


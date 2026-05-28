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
import androidx.compose.ui.unit.sp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateTaskRequest
import com.pudimproductivity.api.Task
import com.pudimproductivity.api.UpdateTaskRequest
import kotlinx.coroutines.launch
import java.time.DayOfWeek
import java.time.LocalDate
import java.time.format.DateTimeFormatter

private val DAY_ORDER = listOf("mon", "tue", "wed", "thu", "fri", "sat", "sun")
private val DAY_SHORT = mapOf(
    "mon" to "M", "tue" to "T", "wed" to "W", "thu" to "T",
    "fri" to "F", "sat" to "S", "sun" to "S"
)

private fun getWeekDates(): List<String> {
    val today = LocalDate.now()
    val monday = today.with(DayOfWeek.MONDAY)
    return (0..6).map { monday.plusDays(it.toLong()).format(DateTimeFormatter.ISO_LOCAL_DATE) }
}

private fun getDayName(dateStr: String): String {
    val date = LocalDate.parse(dateStr)
    val dayIndex = (date.dayOfWeek.value + 6) % 7 // 0=Mon
    return DAY_ORDER[dayIndex]
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TaskListScreen(
    onCreateTask: () -> Unit,
    onTaskClick: (String) -> Unit,
    onListClick: (String) -> Unit
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

    fun loadData() {
        scope.launch {
            isLoading = true
            error = null
            try {
                val todoResult = ApiClient.taskService.listTasks("one-off")
                val habitResult = ApiClient.taskService.listTasks("habit")
                tasks = todoResult + habitResult
                taskLists = ApiClient.taskService.listTaskLists()

                // Load completions for habit tasks
                val weekDates = getWeekDates()
                val from = weekDates.first()
                val to = weekDates.last()
                val map = mutableMapOf<String, List<String>>()
                for (task in habitResult) {
                    try {
                        val completions = ApiClient.taskService.getTaskCompletions(task.id, from, to)
                        map[task.id] = completions.map { it.completed_date }
                    } catch (_: Exception) {
                        map[task.id] = emptyList()
                    }
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

    val todoTasks = tasks.filter { it.recurrence_days == null || it.recurrence_days.isEmpty() }
    val habitTasks = tasks.filter { it.recurrence_days != null && it.recurrence_days.isNotEmpty() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Tasks") },
                actions = {
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
                    // Quick-add
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
                    // Quick-add
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
                                val completedSet = taskCompletions.toSet()
                                val weekDates = getWeekDates()

                                Card(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable { onTaskClick(task.id) },
                                    colors = CardDefaults.cardColors(
                                        containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f)
                                    )
                                ) {
                                    Column(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .padding(12.dp)
                                    ) {
                                        Text(
                                            text = task.title,
                                            style = MaterialTheme.typography.bodyLarge,
                                            modifier = Modifier.weight(1f)
                                        )

                                        Spacer(modifier = Modifier.height(6.dp))

                                        Row(
                                            horizontalArrangement = Arrangement.spacedBy(4.dp)
                                        ) {
                                            weekDates.forEach { date ->
                                                val dayName = getDayName(date)
                                                val isScheduled = task.recurrence_days?.contains(dayName) == true
                                                val isCompleted = date in completedSet
                                                val isToday = date == LocalDate.now().format(DateTimeFormatter.ISO_LOCAL_DATE)

                                                Surface(
                                                    modifier = Modifier.size(28.dp),
                                                    shape = MaterialTheme.shapes.small,
                                                    color = when {
                                                        isCompleted -> MaterialTheme.colorScheme.primary
                                                        isScheduled -> MaterialTheme.colorScheme.tertiaryContainer
                                                        else -> MaterialTheme.colorScheme.surfaceVariant
                                                    },
                                                    contentColor = when {
                                                        isCompleted -> MaterialTheme.colorScheme.onPrimary
                                                        isScheduled -> MaterialTheme.colorScheme.onTertiaryContainer
                                                        else -> MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
                                                    },
                                                    onClick = {
                                                        if (isScheduled || isCompleted) {
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
                                                        }
                                                    }
                                                ) {
                                                    Box(contentAlignment = Alignment.Center) {
                                                        Text(
                                                            text = DAY_SHORT[dayName] ?: "",
                                                            fontSize = 11.sp
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
                selectedTab == 2 -> {
                    // LISTS TAB
                    // Create new list
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

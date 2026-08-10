package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Suggestion
import com.pudimproductivity.api.scheduleService
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DailyPlanScreen() {
    val scope = rememberCoroutineScope()
    var plan by remember { mutableStateOf<Suggestion?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(Unit) {
        try {
            plan = ApiClient.scheduleService.getDailySchedule()
            error = null
        } catch (e: Exception) {
            error = e.message ?: "Failed to load daily plan"
        } finally {
            loading = false
        }
    }

    fun timeLabel(hhmm: String): String {
        val parts = hhmm.split(":")
        val h = parts.getOrNull(0)?.toIntOrNull() ?: 0
        val m = parts.getOrNull(1) ?: "00"
        val suffix = if (h >= 12) "pm" else "am"
        val hour = ((h + 11) % 12) + 1
        return "$hour:$m $suffix"
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Daily Plan") },
                actions = {
                    plan?.let {
                        AssistChip(
                            onClick = {},
                            label = { Text("${("%.1f").format(it.avg_per_day)}/day · ${it.free_hours}h free") }
                        )
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
            when {
                loading -> CircularProgressIndicator()
                error != null -> Text(
                    error!!,
                    color = MaterialTheme.colorScheme.error
                )
                plan != null && plan!!.slots.isEmpty() -> Text(
                    "Nothing to schedule — no pending tasks or habits for today.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                plan != null -> LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    items(plan!!.slots) { slot ->
                        val isHabit = slot.kind == "habit"
                        Card(colors = CardDefaults.cardColors(
                            containerColor = if (isHabit) Color(0xFFF3E8FF) else MaterialTheme.colorScheme.surfaceVariant
                        )) {
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                modifier = Modifier.fillMaxWidth().padding(12.dp)
                            ) {
                                Column(modifier = Modifier.width(110.dp)) {
                                    Text(timeLabel(slot.start_time), style = MaterialTheme.typography.titleSmall)
                                    Text(timeLabel(slot.end_time), style = MaterialTheme.typography.bodySmall)
                                }
                                Column(modifier = Modifier.weight(1f).padding(start = 12.dp)) {
                                    Text(slot.title, style = MaterialTheme.typography.titleSmall)
                                    Text(
                                        slot.kind,
                                        style = MaterialTheme.typography.labelSmall,
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

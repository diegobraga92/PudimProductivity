package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.InsightReport
import com.pudimproductivity.api.insightsService
import kotlinx.coroutines.launch

/**
 * Phase 9a: weekly productivity report — completions, focus minutes, top habits
 * and new recipes, plus an optional LLM summary (insights.llm_enabled flag).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InsightsScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var report by remember { mutableStateOf<InsightReport?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    LaunchedEffect(Unit) {
        try {
            report = ApiClient.insightsService.getWeeklyInsights()
            error = null
        } catch (e: Exception) {
            error = e.message ?: "Failed to load insights"
        } finally {
            loading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("🧠 Insights") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            when {
                loading -> Text("Crunching the numbers…")
                error != null -> Text(error ?: "", color = MaterialTheme.colorScheme.error)
                report == null -> Text("No report available yet.")
                else -> {
                    val stats = report!!.stats

                    // Week badge
                    Text(
                        "Week of ${report!!.week_start}",
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    // Stats cards
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        StatCard("Completed", "${stats.total_completions}", "${"%.1f".format(stats.completions_per_day)}/day", Modifier.weight(1f))
                        StatCard("Focus", "${stats.focus_minutes}m", "${stats.focus_sessions} sessions", Modifier.weight(1f))
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        StatCard("Recipes", "${stats.recipes_created}", "added", Modifier.weight(1f))
                        StatCard("Top habits", "${stats.top_habits?.size ?: 0}", stats.top_habits?.joinToString { it.title }.orEmpty(), Modifier.weight(1f))
                    }

                    // Optional LLM summary
                    report!!.llm_summary?.let { summary ->
                        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)) {
                            Column(Modifier.padding(12.dp)) {
                                Text("✨ AI Coach", style = MaterialTheme.typography.titleSmall)
                                Spacer(Modifier.height(4.dp))
                                Text(summary, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }

                    // Template report
                    Card {
                        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text("📈 Weekly report", style = MaterialTheme.typography.titleSmall)
                            report!!.report_text.split("\n").forEach { line ->
                                if (line.isNotBlank()) {
                                    Text(line.trim(), style = MaterialTheme.typography.bodySmall)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun StatCard(label: String, value: String, sub: String, modifier: Modifier = Modifier) {
    Card(modifier = modifier) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally
        ) {
            Text(value, style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.primary)
            Text(label, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (sub.isNotBlank()) {
                Text(sub, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.outline)
            }
        }
    }
}

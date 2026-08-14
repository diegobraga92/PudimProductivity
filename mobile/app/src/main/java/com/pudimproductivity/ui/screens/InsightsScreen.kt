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
import com.pudimproductivity.i18n.Localization
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
            error = e.message ?: Localization.text("mobile.insights.load.failed")
        } finally {
            loading = false
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(Localization.text("insights.title")) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = Localization.text("common.back"))
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
                loading -> Text(Localization.text("insights.crunching"))
                error != null -> Text(error ?: "", color = MaterialTheme.colorScheme.error)
                report == null -> Text(Localization.text("insights.empty"))
                else -> {
                    val stats = report!!.stats

                    // Week badge
                    Text(
                        Localization.text("insights.weekOf", "date" to report!!.week_start),
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    // Stats cards
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        StatCard(
                            Localization.text("mobile.insights.completed"),
                            "${stats.total_completions}",
                            Localization.text("mobile.insights.perDay", "count" to "%.1f".format(stats.completions_per_day)),
                            Modifier.weight(1f)
                        )
                        StatCard(
                            Localization.text("mobile.insights.focus"),
                            "${stats.focus_minutes}m",
                            Localization.text("mobile.insights.focusSessions", "count" to stats.focus_sessions),
                            Modifier.weight(1f)
                        )
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        StatCard(Localization.text("mobile.insights.recipes"), "${stats.recipes_created}", Localization.text("mobile.insights.added"), Modifier.weight(1f))
                        StatCard(Localization.text("insights.topHabits"), "${stats.top_habits?.size ?: 0}", stats.top_habits?.joinToString { it.title }.orEmpty(), Modifier.weight(1f))
                    }

                    // Optional LLM summary
                    report!!.llm_summary?.let { summary ->
                        Card(colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)) {
                            Column(Modifier.padding(12.dp)) {
                                Text(Localization.text("insights.aiCoach"), style = MaterialTheme.typography.titleSmall)
                                Spacer(Modifier.height(4.dp))
                                Text(summary, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }

                    // Template report
                    Card {
                        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(Localization.text("insights.weeklyReport"), style = MaterialTheme.typography.titleSmall)
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

package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateMealPlanRequest
import com.pudimproductivity.api.MealPlan
import com.pudimproductivity.api.mealPlanService
import kotlinx.coroutines.launch
import java.time.LocalDate

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MealPlanScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var plans by remember { mutableStateOf<List<MealPlan>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var name by remember { mutableStateOf("") }
    var creating by remember { mutableStateOf(false) }
    var expanded by remember { mutableStateOf<String?>(null) }

    suspend fun refresh() {
        try {
            plans = ApiClient.mealPlanService.listMealPlans()
            error = null
        } catch (e: Exception) {
            error = e.message ?: "Failed to load meal plans"
        }
    }

    LaunchedEffect(Unit) {
        scope.launch { refresh(); isLoading = false }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("🗓 Meal Plans") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } }) }
    ) { padding ->
        Column(Modifier.padding(padding).padding(16.dp)) {
            if (creating) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Plan name") }, modifier = Modifier.fillMaxWidth())
                Button(
                    enabled = name.isNotBlank(),
                    onClick = {
                        scope.launch {
                            val today = LocalDate.now()
                            try {
                                ApiClient.mealPlanService.createMealPlan(
                                    CreateMealPlanRequest(name = name.trim(), start_date = today.toString(), end_date = today.plusDays(6).toString())
                                )
                                name = ""
                                creating = false
                                refresh()
                            } catch (e: Exception) {
                                error = e.message ?: "Failed to create plan"
                            }
                        }
                    },
                    modifier = Modifier.fillMaxWidth()
                ) { Text("Create") }
            } else {
                Button(onClick = { creating = true }, modifier = Modifier.fillMaxWidth()) { Text("+ New plan") }
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

            if (isLoading) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            } else {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxSize()) {
                    items(plans) { p ->
                        Card(Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(12.dp)) {
                                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                                    Text(p.name ?: "Untitled plan", style = MaterialTheme.typography.titleMedium)
                                    Text(if (p.is_published) "✓ Published" else "Draft", style = MaterialTheme.typography.labelSmall)
                                }
                                Text("${p.start_date} → ${p.end_date}", style = MaterialTheme.typography.bodySmall)
                                if (expanded == p.id) {
                                    p.slots?.takeIf { it.isNotEmpty() }?.forEach { slot ->
                                        Text("${slot.date} ${slot.meal_type}: recipe ${slot.recipe_id ?: "—"}", style = MaterialTheme.typography.labelSmall)
                                    }
                                    Button(onClick = {
                                        scope.launch {
                                            try {
                                                ApiClient.mealPlanService.generateShoppingList(p.id)
                                            } catch (e: Exception) {
                                                error = e.message ?: "Failed"
                                            }
                                        }
                                    }, modifier = Modifier.fillMaxWidth()) { Text("Generate shopping list") }
                                }
                                TextButton(onClick = { expanded = if (expanded == p.id) null else p.id }) {
                                    Text(if (expanded == p.id) "Collapse" else "Details")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

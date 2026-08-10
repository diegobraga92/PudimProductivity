package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Recipe
import com.pudimproductivity.api.recipeService
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RecipeListScreen(onNew: () -> Unit, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var recipes by remember { mutableStateOf<List<Recipe>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                recipes = ApiClient.recipeService.listRecipes()
            } catch (e: Exception) {
                error = e.message ?: "Failed to load recipes"
            }
            isLoading = false
        }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("🍳 Recipes") }, navigationIcon = { IconButton(onClick = onBack) { Text("←") } }) },
        floatingActionButton = { FloatingActionButton(onClick = onNew) { Text("+") } }
    ) { padding ->
        if (isLoading) {
            Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
        } else if (error != null) {
            Text(error ?: "", modifier = Modifier.padding(24.dp), color = MaterialTheme.colorScheme.error)
        } else {
            LazyColumn(Modifier.fillMaxSize().padding(padding), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(recipes) { r ->
                    Card(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(12.dp)) {
                            Text(r.title, style = MaterialTheme.typography.titleMedium)
                            r.description?.takeIf { it.isNotEmpty() }?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                            Text(
                                "${r.difficulty} · ${r.prep_time_minutes + r.cook_time_minutes} min · ${r.servings} serv",
                                style = MaterialTheme.typography.labelSmall
                            )
                            r.tags?.takeIf { it.isNotEmpty() }?.let { Text(it.joinToString(" ") { "#$it" }, style = MaterialTheme.typography.labelSmall) }
                        }
                    }
                }
            }
        }
    }
}

package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Recipe
import com.pudimproductivity.api.recipeService
import kotlinx.coroutines.launch

/**
 * Read-only recipe detail: loads the full recipe (ingredients, steps, tags)
 * via GET /recipes/{id} and shows it, mirroring the web's RecipeDetail page.
 * Includes a delete action backed by DELETE /recipes/{id}.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RecipeDetailScreen(
    recipeId: String,
    onBack: () -> Unit,
    onDeleted: () -> Unit
) {
    val scope = rememberCoroutineScope()
    var recipe by remember { mutableStateOf<Recipe?>(null) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var showDeleteConfirm by remember { mutableStateOf(false) }

    fun load() {
        scope.launch {
            isLoading = true
            error = null
            try {
                recipe = ApiClient.recipeService.getRecipe(recipeId)
            } catch (e: Exception) {
                error = e.message ?: "Failed to load recipe"
            }
            isLoading = false
        }
    }

    LaunchedEffect(recipeId) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(recipe?.title ?: "Recipe") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    if (recipe != null) {
                        TextButton(onClick = { showDeleteConfirm = true }) {
                            Text("Delete", color = MaterialTheme.colorScheme.error)
                        }
                    }
                }
            )
        }
    ) { padding ->
        when {
            isLoading -> {
                Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            }
            error != null -> {
                Text(
                    error ?: "",
                    modifier = Modifier.padding(24.dp),
                    color = MaterialTheme.colorScheme.error
                )
            }
            recipe != null -> {
                val r = recipe!!
                LazyColumn(
                    Modifier.fillMaxSize().padding(padding).padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    item {
                        Text(
                            text = "${r.difficulty} · ${r.prep_time_minutes + r.cook_time_minutes} min · ${r.servings} serv",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                    }

                    r.description?.takeIf { it.isNotEmpty() }?.let { description ->
                        item {
                            Text(description, style = MaterialTheme.typography.bodyMedium)
                        }
                    }

                    r.tags?.takeIf { it.isNotEmpty() }?.let { tags ->
                        item {
                            Text(
                                tags.joinToString(" ") { "#$it" },
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }
                    }

                    val ingredients = r.ingredients.orEmpty()
                    if (ingredients.isNotEmpty()) {
                        item {
                            Text(
                                "Ingredients",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.SemiBold
                            )
                        }
                        items(ingredients) { ing ->
                            Text(
                                text = listOf(ing.quantity, ing.unit, ing.name)
                                    .filter { it.isNotBlank() }
                                    .joinToString(" "),
                                style = MaterialTheme.typography.bodyLarge
                            )
                        }
                    }

                    val steps = r.steps.orEmpty()
                    if (steps.isNotEmpty()) {
                        item {
                            Text(
                                "Steps",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.SemiBold
                            )
                        }
                        itemsIndexed(steps) { index, step ->
                            val number = step.step_number ?: (index + 1)
                            Row {
                                Text(
                                    "$number.",
                                    style = MaterialTheme.typography.bodyLarge,
                                    fontWeight = FontWeight.SemiBold,
                                    modifier = Modifier.width(28.dp)
                                )
                                Text(
                                    step.instruction,
                                    style = MaterialTheme.typography.bodyLarge,
                                    modifier = Modifier.weight(1f)
                                )
                            }
                        }
                    }
                }
            }
        }
    }

    if (showDeleteConfirm) {
        AlertDialog(
            onDismissRequest = { showDeleteConfirm = false },
            title = { Text("Delete recipe?") },
            text = { Text("This cannot be undone.") },
            confirmButton = {
                TextButton(onClick = {
                    showDeleteConfirm = false
                    scope.launch {
                        try {
                            ApiClient.recipeService.deleteRecipe(recipeId)
                        } catch (_: Exception) {
                            // Best effort; still leave the screen.
                        }
                        onDeleted()
                    }
                }) {
                    Text("Delete", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteConfirm = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}

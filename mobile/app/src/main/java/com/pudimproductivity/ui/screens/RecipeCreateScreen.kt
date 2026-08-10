package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateRecipeRequest
import com.pudimproductivity.api.RecipeIngredient
import com.pudimproductivity.api.recipeService
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RecipeCreateScreen(onDone: () -> Unit) {
    val scope = rememberCoroutineScope()
    var title by remember { mutableStateOf("") }
    var difficulty by remember { mutableStateOf("easy") }
    var ingredient by remember { mutableStateOf("") }
    var ingredients by remember { mutableStateOf<List<RecipeIngredient>>(emptyList()) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    Scaffold(
        topBar = { TopAppBar(title = { Text("New recipe") }, navigationIcon = { IconButton(onClick = onDone) { Text("←") } }) }
    ) { padding ->
        Column(Modifier.padding(padding).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            OutlinedTextField(value = title, onValueChange = { title = it }, label = { Text("Title") }, modifier = Modifier.fillMaxWidth())
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("Difficulty: ", style = MaterialTheme.typography.labelLarge)
                listOf("easy", "medium", "hard").forEach { d ->
                    FilterChip(
                        selected = difficulty == d,
                        onClick = { difficulty = d },
                        label = { Text(d) },
                        modifier = Modifier.padding(end = 4.dp)
                    )
                }
            }
            OutlinedTextField(value = ingredient, onValueChange = { ingredient = it }, label = { Text("Ingredient: name, qty, unit") }, modifier = Modifier.fillMaxWidth())
            Button(
                onClick = {
                    val parts = ingredient.split(",").map { it.trim() }
                    if (parts.isNotEmpty() && parts[0].isNotEmpty()) {
                        ingredients = ingredients + RecipeIngredient(name = parts[0], quantity = parts.getOrNull(1) ?: "", unit = parts.getOrNull(2) ?: "")
                        ingredient = ""
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) { Text("+ Add ingredient") }
            if (ingredients.isNotEmpty()) {
                Text("Ingredients: ${ingredients.joinToString("; ") { "${it.name} ${it.quantity} ${it.unit}".trim() }}", style = MaterialTheme.typography.bodySmall)
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error, textAlign = TextAlign.Center) }
            Button(
                enabled = title.isNotBlank() && !saving,
                onClick = {
                    scope.launch {
                        saving = true
                        try {
                            ApiClient.recipeService.createRecipe(
                                CreateRecipeRequest(
                                    title = title.trim(),
                                    difficulty = difficulty,
                                    servings = 1,
                                    ingredients = ingredients
                                )
                            )
                            onDone()
                        } catch (e: Exception) {
                            error = e.message ?: "Failed to create recipe"
                        }
                        saving = false
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) { Text(if (saving) "Saving…" else "Create recipe") }
        }
    }
}

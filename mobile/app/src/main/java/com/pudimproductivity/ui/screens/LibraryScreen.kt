package com.pudimproductivity.ui.screens

import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.CreateLibraryItemRequest
import com.pudimproductivity.api.LibraryItem
import com.pudimproductivity.api.UpdateLibraryItemRequest
import com.pudimproductivity.api.libraryService
import com.pudimproductivity.i18n.Localization
import kotlinx.coroutines.launch

private val MEDIA_TYPES = listOf("movie", "series", "book", "game")

private fun mediaLabel(t: String): String = when (t) {
    "movie" -> "🎬 " + Localization.text("library.movie")
    "series" -> "📺 " + Localization.text("library.series")
    "book" -> "📚 " + Localization.text("library.book")
    "game" -> "🎮 " + Localization.text("library.game")
    else -> t
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LibraryScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var items by remember { mutableStateOf<List<LibraryItem>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    var formOpen by remember { mutableStateOf(false) }
    var editing by remember { mutableStateOf<LibraryItem?>(null) }
    var name by remember { mutableStateOf("") }
    var mediaType by remember { mutableStateOf("book") }
    var year by remember { mutableStateOf("") }
    var done by remember { mutableStateOf(false) }
    var notes by remember { mutableStateOf("") }
    var score by remember { mutableStateOf("") }
    var scoreSource by remember { mutableStateOf("") }

    suspend fun refresh() {
        try {
            items = ApiClient.libraryService.listItems()
            error = null
        } catch (e: Exception) {
            error = e.message ?: Localization.text("mobile.library.load.failed")
        }
    }

    LaunchedEffect(Unit) {
        scope.launch { refresh(); isLoading = false }
    }

    fun openAdd() {
        editing = null
        name = ""
        mediaType = "book"
        year = ""
        done = false
        notes = ""
        score = ""
        scoreSource = ""
        formOpen = true
    }

    fun openEdit(item: LibraryItem) {
        editing = item
        name = item.name
        mediaType = item.media_type
        year = item.release_year?.toString() ?: ""
        done = item.done
        notes = item.notes
        score = item.score?.toString() ?: ""
        scoreSource = item.score_source
        formOpen = true
    }

    fun closeForm() {
        formOpen = false
        editing = null
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(Localization.text("library.title")) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = Localization.text("common.back"))
                    }
                }
            )
        }
    ) { padding ->
        Column(Modifier.padding(padding).padding(16.dp)) {

            if (formOpen) {
                Card(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
                    Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(
                            value = name,
                            onValueChange = { name = it },
                            label = { Text(Localization.text("common.name")) },
                            modifier = Modifier.fillMaxWidth()
                        )
                        Row(Modifier.horizontalScroll(rememberScrollState())) {
                            MEDIA_TYPES.forEach { t ->
                                FilterChip(
                                    selected = mediaType == t,
                                    onClick = { mediaType = t },
                                    label = { Text(Localization.text("library.$t")) },
                                    modifier = Modifier.padding(end = 8.dp)
                                )
                            }
                        }
                        OutlinedTextField(
                            value = year,
                            onValueChange = { year = it.filter { c -> c.isDigit() } },
                            label = { Text(Localization.text("mobile.library.releaseYear")) },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                            modifier = Modifier.fillMaxWidth()
                        )
                        OutlinedTextField(
                            value = score,
                            onValueChange = { score = it.filter { c -> c.isDigit() || c == '.' } },
                            label = { Text(Localization.text("mobile.library.scoreLabel")) },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            modifier = Modifier.fillMaxWidth()
                        )
                        OutlinedTextField(
                            value = scoreSource,
                            onValueChange = { scoreSource = it },
                            label = { Text(Localization.text("mobile.library.sourceLabel")) },
                            modifier = Modifier.fillMaxWidth()
                        )
                        OutlinedTextField(
                            value = notes,
                            onValueChange = { notes = it },
                            label = { Text(Localization.text("common.notes")) },
                            modifier = Modifier.fillMaxWidth()
                        )
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Checkbox(checked = done, onCheckedChange = { done = it })
                            Text(Localization.text("library.doneLabel"))
                        }
                        Button(
                            enabled = name.isNotBlank(),
                            onClick = {
                                scope.launch {
                                    try {
                                        val yearValue = year.toIntOrNull()?.takeIf { it in 1800..2100 }
                                        val scoreValue = score.toDoubleOrNull()?.takeIf { it in 0.0..100.0 }
                                        if (editing != null) {
                                            ApiClient.libraryService.updateItem(
                                                editing!!.id,
                                                UpdateLibraryItemRequest(
                                                    name = name.trim(),
                                                    media_type = mediaType,
                                                    release_year = yearValue,
                                                    done = done,
                                                    notes = notes,
                                                    score = scoreValue,
                                                    score_source = scoreSource.trim()
                                                )
                                            )
                                        } else {
                                            ApiClient.libraryService.createItem(
                                                CreateLibraryItemRequest(
                                                    name = name.trim(),
                                                    media_type = mediaType,
                                                    release_year = yearValue,
                                                    done = done,
                                                    notes = notes,
                                                    score = scoreValue,
                                                    score_source = scoreSource.trim()
                                                )
                                            )
                                        }
                                        closeForm()
                                        refresh()
                                    } catch (e: Exception) {
                                        error = e.message ?: Localization.text("mobile.library.save.failed")
                                    }
                                }
                            },
                            modifier = Modifier.fillMaxWidth()
                        ) { Text(if (editing != null) Localization.text("library.saveChanges") else Localization.text("common.add")) }
                    }
                }
            }

            error?.let {
                Text(
                    it,
                    color = MaterialTheme.colorScheme.error,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.fillMaxWidth()
                )
            }

            Button(
                onClick = { if (formOpen) closeForm() else openAdd() },
                modifier = Modifier.fillMaxWidth()
            ) { Text(if (formOpen) Localization.text("common.cancel") else Localization.text("library.addItem")) }

            if (isLoading) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            } else {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxSize()) {
                    items(items) { item ->
                        Card(Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                Row(verticalAlignment = Alignment.CenterVertically) {
                                    Column(Modifier.weight(1f)) {
                                        Text(item.name, style = MaterialTheme.typography.titleMedium)
                                        Text(
                                            "${mediaLabel(item.media_type)}${item.release_year?.let { " · $it" } ?: ""}",
                                            style = MaterialTheme.typography.bodySmall
                                        )
                                        item.score?.let { scoreVal ->
                                            Text(
                                                "★ $scoreVal${if (item.score_source.isNotBlank()) " (${item.score_source})" else ""}",
                                                style = MaterialTheme.typography.bodySmall
                                            )
                                        }
                                    }
                                    Checkbox(
                                        checked = item.done,
                                        onCheckedChange = { checked ->
                                            scope.launch {
                                                try {
                                                    ApiClient.libraryService.updateItem(
                                                        item.id,
                                                        UpdateLibraryItemRequest(done = checked)
                                                    )
                                                    refresh()
                                                } catch (e: Exception) {
                                                    error = e.message ?: Localization.text("mobile.library.update.failed")
                                                }
                                            }
                                        }
                                    )
                                }
                                if (item.notes.isNotBlank()) {
                                    Text("📝 ${item.notes}", style = MaterialTheme.typography.bodySmall)
                                }
                                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                    OutlinedButton(onClick = { openEdit(item) }, modifier = Modifier.weight(1f)) {
                                        Text(Localization.text("common.edit"))
                                    }
                                    OutlinedButton(
                                        onClick = {
                                            scope.launch {
                                                try {
                                                    ApiClient.libraryService.deleteItem(item.id)
                                                    refresh()
                                                } catch (e: Exception) {
                                                    error = e.message ?: Localization.text("mobile.library.delete.failed")
                                                }
                                            }
                                        },
                                        modifier = Modifier.weight(1f)
                                    ) { Text(Localization.text("common.delete")) }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}


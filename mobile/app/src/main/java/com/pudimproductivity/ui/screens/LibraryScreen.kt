package com.pudimproductivity.ui.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
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

internal fun formatScore(score: Double): String {
    val rounded = Math.round(score * 10) / 10.0
    return if (rounded % 1.0 == 0.0) rounded.toLong().toString() else rounded.toString()
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
    var subtype by remember { mutableStateOf("") }
    var subtypeOptions by remember { mutableStateOf<List<String>>(emptyList()) }
    var subtypeFilter by remember { mutableStateOf<String?>(null) }
    var filterMenuOpen by remember { mutableStateOf(false) }

    suspend fun refresh() {
        try {
            items = ApiClient.libraryService.listItems(subtype = subtypeFilter)
            subtypeOptions = ApiClient.libraryService.subtypes()
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
        subtype = ""
        formOpen = true
    }

    fun openEdit(item: LibraryItem) {
        editing = item
        name = item.name
        mediaType = item.media_type
        year = item.release_year?.toString() ?: ""
        done = item.done
        notes = item.notes
        score = item.score?.let { formatScore(it) } ?: ""
        scoreSource = item.score_source
        subtype = item.subtype
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
                            value = subtype,
                            onValueChange = { subtype = it },
                            label = { Text(Localization.text("common.subtype")) },
                            placeholder = {
                                Text(
                                    Localization.text(
                                        if (mediaType == "game") "library.consolePlaceholder" else "library.genrePlaceholder"
                                    )
                                )
                            },
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
                                        val scoreValue = score.toDoubleOrNull()
                                            ?.takeIf { it in 0.0..100.0 }
                                            ?.let { Math.round(it * 10) / 10.0 }
                                        if (editing != null) {
                                            ApiClient.libraryService.updateItem(
                                                editing!!.id,
                                                UpdateLibraryItemRequest(
                                                    name = name.trim(),
                                                    media_type = mediaType,
                                                    release_year = yearValue,
                                                    done = done,
                                                    notes = notes,
                                                    subtype = subtype.trim(),
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
                                                    subtype = subtype.trim(),
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

            // Subtype (genre/console) filter.
            Box(Modifier.fillMaxWidth().padding(vertical = 8.dp)) {
                OutlinedButton(
                    onClick = { filterMenuOpen = true },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text(subtypeFilter ?: Localization.text("library.filterSubtypeAll"))
                }
                DropdownMenu(expanded = filterMenuOpen, onDismissRequest = { filterMenuOpen = false }) {
                    DropdownMenuItem(
                        text = { Text(Localization.text("library.filterSubtypeAll")) },
                        onClick = {
                            subtypeFilter = null
                            filterMenuOpen = false
                            scope.launch { refresh() }
                        }
                    )
                    subtypeOptions.forEach { option ->
                        DropdownMenuItem(
                            text = { Text(option) },
                            onClick = {
                                subtypeFilter = option
                                filterMenuOpen = false
                                scope.launch { refresh() }
                            }
                        )
                    }
                }
            }

            Button(
                onClick = { if (formOpen) closeForm() else openAdd() },
                modifier = Modifier.fillMaxWidth()
            ) { Text(if (formOpen) Localization.text("common.cancel") else Localization.text("library.addItem")) }

            if (isLoading) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            } else if (items.isEmpty()) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                    Text(
                        text = Localization.text("mobile.library.empty"),
                        style = MaterialTheme.typography.bodyLarge,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            } else {
                LibraryRowList(
                    items = items,
                    onToggleDone = { item, checked ->
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
                    },
                    onEdit = { openEdit(it) },
                    onDelete = { item ->
                        scope.launch {
                            try {
                                ApiClient.libraryService.deleteItem(item.id)
                                refresh()
                            } catch (e: Exception) {
                                error = e.message ?: Localization.text("mobile.library.delete.failed")
                            }
                        }
                    }
                )
            }
        }
    }
}

@Composable
private fun LibraryRowList(
    items: List<LibraryItem>,
    onToggleDone: (LibraryItem, Boolean) -> Unit,
    onEdit: (LibraryItem) -> Unit,
    onDelete: (LibraryItem) -> Unit
) {
    val borderColor = MaterialTheme.colorScheme.outlineVariant
    val shape = RoundedCornerShape(8.dp)
    // Fixed table width so columns line up. Scrolls horizontally on narrow screens.
    val tableWidth = 780.dp

    Row(
        modifier = Modifier
            .fillMaxSize()
            .clip(shape)
            .border(1.dp, borderColor, shape)
            .horizontalScroll(rememberScrollState())
    ) {
        LazyColumn(modifier = Modifier.width(tableWidth).fillMaxHeight()) {
            // Header row
            item {
                Row(
                    modifier = Modifier
                        .width(tableWidth)
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f))
                        .border(1.dp, borderColor.copy(alpha = 0.6f))
                        .padding(horizontal = 12.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = Localization.text("common.name"),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.weight(1f)
                    )
                    Text(
                        text = Localization.text("common.type"),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.width(110.dp)
                    )
                    Text(
                        text = Localization.text("common.subtype"),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.width(140.dp)
                    )
                    Text(
                        text = Localization.text("common.year"),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.width(60.dp)
                    )
                    Text(
                        text = Localization.text("common.score"),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.width(72.dp)
                    )
                    Text(
                        text = Localization.text("common.actions"),
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.SemiBold,
                        modifier = Modifier.width(150.dp)
                    )
                }
            }

            items(items) { item ->
                Row(
                    modifier = Modifier
                        .width(tableWidth)
                        .border(1.dp, borderColor.copy(alpha = 0.4f))
                        .padding(horizontal = 12.dp, vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Checkbox(
                                checked = item.done,
                                onCheckedChange = { onToggleDone(item, it) },
                                modifier = Modifier.padding(end = 4.dp)
                            )
                            Text(
                                text = item.name,
                                style = MaterialTheme.typography.bodyMedium,
                                fontWeight = FontWeight.Medium,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                                modifier = Modifier.weight(1f)
                            )
                        }
                        if (item.notes.isNotBlank()) {
                            Text(
                                text = "📝 ${item.notes}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                                modifier = Modifier.padding(start = 28.dp)
                            )
                        }
                    }

                    Text(
                        text = mediaLabel(item.media_type),
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.width(110.dp)
                    )
                    Text(
                        text = item.subtype.ifBlank { "—" },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.width(140.dp)
                    )
                    Text(
                        text = item.release_year?.toString() ?: "—",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.width(60.dp)
                    )
                    Text(
                        text = item.score?.let { "★ ${formatScore(it)}" } ?: "—",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.width(72.dp)
                    )
                    Row(
                        modifier = Modifier.width(170.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        OutlinedButton(
                            onClick = { onEdit(item) },
                            modifier = Modifier.weight(1f),
                            contentPadding = PaddingValues(horizontal = 12.dp)
                        ) {
                            Text(
                                Localization.text("common.edit"),
                                style = MaterialTheme.typography.labelSmall,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis
                            )
                        }
                        OutlinedButton(
                            onClick = { onDelete(item) },
                            modifier = Modifier.weight(1f),
                            contentPadding = PaddingValues(horizontal = 12.dp)
                        ) {
                            Text(
                                Localization.text("common.delete"),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.error,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis
                            )
                        }
                    }
                }
            }
        }
    }
}

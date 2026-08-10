package com.pudimproductivity.ui.screens

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.AddByISBNRequest
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Book
import com.pudimproductivity.api.bookService
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BookListScreen(onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    var books by remember { mutableStateOf<List<Book>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var isbnInput by remember { mutableStateOf("") }

    suspend fun refresh() {
        try {
            books = ApiClient.bookService.listBooks()
            error = null
        } catch (e: Exception) {
            error = e.message ?: "Failed to load books"
        }
    }

    LaunchedEffect(Unit) {
        scope.launch { refresh(); isLoading = false }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("📚 Books") }, navigationIcon = { IconButton(onClick = onBack) { Text("←") } }) }
    ) { padding ->
        Column(Modifier.padding(padding).padding(16.dp)) {
            OutlinedTextField(value = isbnInput, onValueChange = { isbnInput = it }, label = { Text("ISBN") }, modifier = Modifier.fillMaxWidth())
            Button(
                enabled = isbnInput.isNotBlank(),
                onClick = {
                    scope.launch {
                        try {
                            ApiClient.bookService.addByISBN(AddByISBNRequest(isbnInput.trim()))
                            isbnInput = ""
                            refresh()
                        } catch (e: Exception) {
                            error = e.message ?: "Lookup failed"
                        }
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) { Text("Add by ISBN") }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error, textAlign = TextAlign.Center, modifier = Modifier.fillMaxWidth()) }

            if (isLoading) {
                Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            } else {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxSize()) {
                    items(books) { b ->
                        Card(Modifier.fillMaxWidth()) {
                            Row(Modifier.padding(12.dp)) {
                                Column(Modifier.weight(1f)) {
                                    Text(b.title, style = MaterialTheme.typography.titleMedium)
                                    Text(b.authors.joinToString(", ").ifEmpty { "Unknown author" }, style = MaterialTheme.typography.bodySmall)
                                    Text("${b.status} · ${b.page_count} pp", style = MaterialTheme.typography.labelSmall)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

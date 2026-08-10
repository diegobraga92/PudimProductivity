package com.pudimproductivity.ui.screens

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import com.pudimproductivity.api.AddByISBNRequest
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.Book
import com.pudimproductivity.api.bookService
import com.pudimproductivity.api.imagePart
import com.pudimproductivity.api.mediaService
import kotlinx.coroutines.launch
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BookListScreen(onBack: () -> Unit) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val scope = rememberCoroutineScope()
    var books by remember { mutableStateOf<List<Book>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var isbnInput by remember { mutableStateOf("") }
    var scanning by remember { mutableStateOf(false) }

    suspend fun refresh() {
        try {
            books = ApiClient.bookService.listBooks()
            error = null
        } catch (e: Exception) {
            error = e.message ?: "Failed to load books"
        }
    }

    // Phase 9b: camera → scan-isbn → auto-fill the ISBN field.
    val scanLauncher = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { success ->
        if (success) {
            val file = File(context.cacheDir, "isbn_scan.jpg")
            if (file.exists()) {
                scanning = true
                scope.launch {
                    try {
                        val result = ApiClient.mediaService.scanIsbn(imagePart(file))
                        isbnInput = result.isbn
                        // Auto-add by ISBN once decoded.
                        ApiClient.bookService.addByISBN(AddByISBNRequest(result.isbn))
                        isbnInput = ""
                        refresh()
                    } catch (e: Exception) {
                        error = e.message ?: "Barcode scan failed"
                    } finally {
                        scanning = false
                        file.delete()
                    }
                }
            }
        }
    }

    fun launchCamera() {
        val file = File(context.cacheDir, "isbn_scan.jpg")
        val uri: Uri = FileProvider.getUriForFile(
            context,
            "com.pudimproductivity.fileprovider",
            file
        )
        scanLauncher.launch(uri)
    }

    LaunchedEffect(Unit) {
        scope.launch { refresh(); isLoading = false }
    }

    Scaffold(
        topBar = { TopAppBar(title = { Text("📚 Books") }, navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back") } }) }
    ) { padding ->
        Column(Modifier.padding(padding).padding(16.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                OutlinedTextField(value = isbnInput, onValueChange = { isbnInput = it }, label = { Text("ISBN") }, modifier = Modifier.weight(1f))
                // Phase 9b: photograph a book barcode → auto-fill.
                OutlinedButton(
                    onClick = { launchCamera() },
                    enabled = !scanning,
                    modifier = Modifier.align(Alignment.CenterVertically)
                ) { Text(if (scanning) "…" else "📷") }
            }
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

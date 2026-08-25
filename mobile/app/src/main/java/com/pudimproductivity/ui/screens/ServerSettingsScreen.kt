package com.pudimproductivity.ui.screens

import android.content.Context
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Error
import androidx.compose.material.icons.filled.RestartAlt
import androidx.compose.material.icons.filled.Save
import androidx.compose.material.icons.filled.WifiTethering
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pudimproductivity.api.ApiClient
import com.pudimproductivity.api.HealthResponse
import com.pudimproductivity.api.HealthService
import com.pudimproductivity.api.ServerConfig
import com.pudimproductivity.api.SyncClient
import com.pudimproductivity.i18n.AppLanguage
import com.pudimproductivity.i18n.Localization
import com.pudimproductivity.ui.theme.ThemeMode
import kotlinx.coroutines.launch
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory

/**
 * Server settings screen.
 *
 * Lets the user point the app at any PudimProductivity backend without
 * rebuilding: the URL is persisted via [ServerConfig] and applied live
 * (ApiClient's Retrofit is invalidated and the sync WebSocket reconnects).
 */
@Composable
fun ServerSettingsScreen(
    onBack: () -> Unit,
    themeMode: ThemeMode,
    onThemeModeChange: (ThemeMode) -> Unit
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val currentUrl by ServerConfig.url.collectAsState()

    var urlInput by remember { mutableStateOf(currentUrl) }
    var saved by remember { mutableStateOf(false) }
    var testResult by remember { mutableStateOf<TestResult?>(null) }
    var testing by remember { mutableStateOf(false) }

    Column(modifier = Modifier.padding(24.dp)) {
        Text(
            text = Localization.text("settings.server.title"),
            style = MaterialTheme.typography.headlineLarge
        )

        Text(
            text = Localization.text("settings.server.backendUrl"),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = 16.dp)
        )
        Text(
            text = Localization.text("settings.server.description"),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(top = 4.dp)
        )

        OutlinedTextField(
            value = urlInput,
            onValueChange = {
                urlInput = it
                saved = false
                testResult = null
            },
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 12.dp),
            label = { Text(Localization.text("settings.server.apiBaseUrl")) },
            placeholder = { Text(ServerConfig.defaultUrl) },
            singleLine = true,
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
            isError = testResult?.ok == false
        )

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Button(
                onClick = {
                    scope.launch {
                        ServerConfig.setUrl(context, urlInput)
                        applyServerUrl(context)
                        saved = true
                    }
                },
                enabled = urlInput.isNotBlank()
            ) {
                Icon(Icons.Filled.Save, contentDescription = null)
                Spacer(Modifier.width(8.dp))
                Text(Localization.text("common.save"))
            }

            OutlinedButton(
                onClick = {
                    scope.launch {
                        testing = true
                        testResult = null
                        // Test against the value currently typed (not yet saved).
                        try {
                            val resp = testConnection(urlInput)
                            testResult = TestResult(true, "${Localization.text("nav.status")}: ${resp.status} · ${Localization.text("status.database")}: ${resp.db} · v${resp.version}")
                        } catch (e: Exception) {
                            testResult = TestResult(false, e.message ?: Localization.text("settings.server.testFailed"))
                        } finally {
                            testing = false
                        }
                    }
                },
                enabled = urlInput.isNotBlank() && !testing
            ) {
                if (testing) {
                    CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
                } else {
                    Icon(Icons.Filled.WifiTethering, contentDescription = null)
                }
                Spacer(Modifier.width(8.dp))
                Text(Localization.text("settings.server.test"))
            }
        }


        // Saved / test feedback.
        if (saved) {
            Row(
                modifier = Modifier.padding(top = 12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    Icons.Filled.CheckCircle,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    Localization.text("settings.server.saved"),
                    color = MaterialTheme.colorScheme.primary
                )
            }
        }

        testResult?.let { result ->
            Row(
                modifier = Modifier.padding(top = 12.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = if (result.ok) Icons.Filled.CheckCircle else Icons.Filled.Error,
                    contentDescription = null,
                    tint = if (result.ok) MaterialTheme.colorScheme.primary
                    else MaterialTheme.colorScheme.error
                )
                Spacer(Modifier.width(8.dp))
                Text(
                    text = result.message,
                    color = if (result.ok) MaterialTheme.colorScheme.primary
                    else MaterialTheme.colorScheme.error
                )
            }
        }

        // Currently active URL.
        Text(
            text = Localization.text("settings.server.activeUrl"),
            style = MaterialTheme.typography.titleSmall,
            modifier = Modifier.padding(top = 24.dp)
        )
        Text(
            text = currentUrl,
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.padding(top = 4.dp)
        )

        TextButton(
            onClick = {
                scope.launch {
                    ServerConfig.resetToDefault(context)
                    applyServerUrl(context)
                    urlInput = ServerConfig.url.value
                    saved = true
                    testResult = null
                }
            },
            modifier = Modifier.padding(top = 16.dp)
        ) {
            Icon(Icons.Filled.RestartAlt, contentDescription = null)
            Spacer(Modifier.width(8.dp))
            Text(Localization.text("settings.server.reset", "url" to ServerConfig.defaultUrl))
        }

        // Language toggle.
        Text(
            text = Localization.text("lang.language"),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = 24.dp)
        )
        Row(
            modifier = Modifier.padding(top = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            AppLanguage.entries.forEach { lang ->
                FilterChip(
                    selected = Localization.language == lang,
                    onClick = {
                        if (Localization.language != lang) {
                            Localization.setLanguage(context, lang)
                            // Refresh home-screen widgets so they pick up the new language.
                            scope.launch { com.pudimproductivity.widget.WidgetUpdater.updateAll(context) }
                        }
                    },
                    label = { Text(Localization.text(Localization.LANGUAGE_KEYS.getValue(lang))) }
                )
            }
        }

        // Theme selector (System / Light / Dark).
        Text(
            text = Localization.text("settings.theme"),
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = 24.dp)
        )
        Row(
            modifier = Modifier.padding(top = 8.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            ThemeMode.entries.forEach { mode ->
                FilterChip(
                    selected = themeMode == mode,
                    onClick = { onThemeModeChange(mode) },
                    label = {
                        Text(
                            Localization.text(
                                when (mode) {
                                    ThemeMode.SYSTEM -> "settings.theme.system"
                                    ThemeMode.LIGHT -> "settings.theme.light"
                                    ThemeMode.DARK -> "settings.theme.dark"
                                }
                            )
                        )
                    }
                )
            }
        }

        Spacer(modifier = Modifier.height(16.dp))
        Button(onClick = onBack) {
            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
            Spacer(Modifier.width(8.dp))
            Text(Localization.text("common.backToTasks"))
        }
    }
}

/** Applies a persisted URL change to the live API client + sync WebSocket. */
private fun applyServerUrl(context: Context) {
    ApiClient.invalidate()
    SyncClient.stop()
    SyncClient.start(context)
}

private data class TestResult(val ok: Boolean, val message: String)

/** Performs a throwaway health check against [url] without disturbing the client. */
private suspend fun testConnection(url: String): HealthResponse {
    // Build a temporary Retrofit against the candidate URL.
    val tempRetrofit = Retrofit.Builder()
        .baseUrl(url.trimEnd('/') + "/")
        .client(ApiClient.client)
        .addConverterFactory(GsonConverterFactory.create())
        .build()
    return tempRetrofit.create(HealthService::class.java).getHealth()
}


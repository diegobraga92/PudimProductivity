package com.pudimproductivity.api

import android.content.Context
import androidx.core.content.edit
import com.pudimproductivity.BuildConfig
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Runtime-configurable backend base URL.
 *
 * The URL is persisted in SharedPreferences so it survives app restarts and
 * can be changed from the in-app Server Settings screen — no rebuild needed.
 * When no override is stored, the build-time default ([BuildConfig.API_BASE_URL],
 * injected via `buildConfigField` in app/build.gradle.kts) is used.
 *
 * Call [init] once at app startup (MainActivity.onCreate) before any API
 * client is touched.
 */
object ServerConfig {

    private const val PREFS_NAME = "pudim"
    private const val URL_KEY = "api.base.url"

    private val _url = MutableStateFlow(BuildConfig.API_BASE_URL)
    val url: StateFlow<String> = _url.asStateFlow()

    @Volatile
    private var initialized = false

    /** The build-time default (emulator loopback or the -P/CI override). */
    val defaultUrl: String get() = BuildConfig.API_BASE_URL

    /** Loads the persisted override (if any) from SharedPreferences. Idempotent. */
    fun init(context: Context) {
        if (initialized) return
        synchronized(this) {
            if (initialized) return
            val prefs = context.applicationContext
                .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            _url.value = prefs.getString(URL_KEY, null) ?: BuildConfig.API_BASE_URL
            initialized = true
        }
    }

    /**
     * Persists [newUrl] as the active base URL. Pass [BuildConfig.API_BASE_URL]
     * to clear the override and return to the build-time default.
     */
    fun setUrl(context: Context, newUrl: String) {
        val trimmed = newUrl.trim().trimEnd('/')
        if (trimmed.isEmpty()) return
        context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit { putString(URL_KEY, trimmed) }
        _url.value = trimmed
    }

    /** Removes the stored override, reverting to [BuildConfig.API_BASE_URL]. */
    fun resetToDefault(context: Context) {
        context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit { remove(URL_KEY) }
        _url.value = BuildConfig.API_BASE_URL
    }
}

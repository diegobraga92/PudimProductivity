package com.pudimproductivity.ui.theme

import android.content.Context
import androidx.core.content.edit

/** In-app theme choice: follow the system setting, or force light/dark. */
enum class ThemeMode { SYSTEM, LIGHT, DARK }

/**
 * Persists the in-app theme in the same "pudim" SharedPreferences file used by
 * ServerConfig, TaskSortPreferences and LanguagePreferences.
 */
object ThemePreferences {
    private const val PREFS_NAME = "pudim"
    private const val KEY = "theme"

    fun load(context: Context): ThemeMode {
        val stored = context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .getString(KEY, null)
        return ThemeMode.entries.firstOrNull { it.name == stored } ?: ThemeMode.SYSTEM
    }

    fun save(context: Context, mode: ThemeMode) {
        context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit { putString(KEY, mode.name) }
    }
}

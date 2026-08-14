package com.pudimproductivity.utils

import android.content.Context
import androidx.core.content.edit

/**
 * Persists task ordering preferences in the app's SharedPreferences (the same
 * "pudim" file used by ServerConfig), mirroring the web's `taskSort.*`
 * localStorage keys so behavior matches across clients.
 */
object TaskSortPreferences {
    private const val PREFS_NAME = "pudim"

    fun load(context: Context, key: String): SortOption {
        val stored = context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .getString(key, null)
        return SortOption.fromKey(stored)
    }

    fun save(context: Context, key: String, option: SortOption) {
        context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit { putString(key, option.key) }
    }
}

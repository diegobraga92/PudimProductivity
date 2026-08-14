package com.pudimproductivity.i18n

import android.content.Context
import androidx.core.content.edit

/**
 * Persists the in-app UI language in the same "pudim" SharedPreferences file
 * used by ServerConfig and TaskSortPreferences.
 */
object LanguagePreferences {
    private const val PREFS_NAME = "pudim"
    private const val KEY = "language"

    fun load(context: Context): AppLanguage {
        val stored = context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .getString(KEY, null)
        return AppLanguage.fromCode(stored)
    }

    fun save(context: Context, language: AppLanguage) {
        context.applicationContext
            .getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit { putString(KEY, language.code) }
    }
}

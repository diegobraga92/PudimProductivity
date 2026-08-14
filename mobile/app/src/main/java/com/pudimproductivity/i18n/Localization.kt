package com.pudimproductivity.i18n

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken

/**
 * In-app localization backed by the shared dictionary (`shared/i18n/en.json`
 * and `shared/i18n/pt-BR.json`, copied into `assets/i18n/` by Gradle).
 *
 * [language] is Compose state, so any composable that calls [text] recomposes
 * when the user switches languages. Non-UI entry points (widgets, background
 * workers) read the current value directly.
 */
object Localization {

    /** Dictionary key → translation key for language names. */
    val LANGUAGE_KEYS: Map<AppLanguage, String> = mapOf(
        AppLanguage.EN to "lang.english",
        AppLanguage.PT_BR to "lang.portuguese",
    )

    var language: AppLanguage by mutableStateOf(AppLanguage.EN)
        private set

    private val gson = Gson()
    private val dictionaries = mutableMapOf<AppLanguage, Map<String, String>>()
    private var initialized = false

    /**
     * Loads the dictionaries from assets and the persisted language choice.
     * Idempotent; safe to call from every entry point (activity, widget
     * receivers, background workers).
     */
    fun init(context: Context) {
        synchronized(this) {
            if (initialized) return
            for (lang in AppLanguage.entries) {
                val fileName = "i18n/${lang.code}.json"
                val raw = context.assets.open(fileName)
                    .bufferedReader(Charsets.UTF_8)
                    .use { it.readText() }
                val type = object : TypeToken<Map<String, String>>() {}.type
                @Suppress("UNCHECKED_CAST")
                dictionaries[lang] = gson.fromJson(raw, type) as Map<String, String>
            }
            language = LanguagePreferences.load(context)
            initialized = true
        }
    }

    /** Switches the UI language, persists it, and refreshes home-screen widgets. */
    fun setLanguage(context: Context, newLanguage: AppLanguage) {
        if (language == newLanguage) return
        language = newLanguage
        LanguagePreferences.save(context, newLanguage)
    }

    /**
     * Translates a dictionary key with `{name}` interpolation and simple
     * ICU-lite pluralization: `{count, plural, one {…} other {…}}`.
     */
    fun text(key: String, vararg args: Pair<String, Any>): String {
        val template = dictionaries[language]?.get(key)
            ?: dictionaries[AppLanguage.EN]?.get(key)
            ?: key
        return interpolate(template, args.toMap())
    }

    private val monthKeys = listOf(
        "months.jan", "months.feb", "months.mar", "months.apr", "months.may", "months.jun",
        "months.jul", "months.aug", "months.sep", "months.oct", "months.nov", "months.dec"
    )

    /** Localized month abbreviations for date range formatting. */
    fun months(): List<String> = monthKeys.map { text(it) }

    private val pluralRe = Regex("""\{(\w+), plural, one \{(.*?)\} other \{(.*?)\}\}""")
    private val simpleRe = Regex("""\{(\w+)\}""")

    fun interpolate(template: String, args: Map<String, Any>): String {
        val withPlural = pluralRe.replace(template) { match ->
            val key = match.groupValues[1]
            val one = match.groupValues[2]
            val other = match.groupValues[3]
            val count = (args[key] as? Number)?.toLong() ?: 0L
            if (count == 1L) one else other
        }
        return simpleRe.replace(withPlural) { match ->
            args[match.groupValues[1]]?.toString() ?: match.value
        }
    }
}

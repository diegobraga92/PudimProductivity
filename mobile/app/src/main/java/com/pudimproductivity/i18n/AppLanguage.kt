package com.pudimproductivity.i18n

/**
 * UI languages supported by the app. Codes match the shared dictionary files
 * in `shared/i18n/{code}.json`.
 */
enum class AppLanguage(val code: String) {
    EN("en"),
    PT_BR("pt-BR");

    companion object {
        /** Resolves a persisted/serialized code, falling back to English. */
        fun fromCode(code: String?): AppLanguage =
            entries.firstOrNull { it.code.equals(code, ignoreCase = true) } ?: EN
    }
}

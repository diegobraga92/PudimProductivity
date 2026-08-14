import en from "../../../shared/i18n/en.json";
import ptBR from "../../../shared/i18n/pt-BR.json";

/**
 * Supported UI languages. The dictionary lives in `shared/i18n/` so the web
 * (this module) and the Android app (assets copied by Gradle) share one
 * single source of truth.
 */
export type Language = "en" | "pt-BR";

/** Flat key → translated string map. */
export type Dictionary = Record<string, string>;

export const dictionaries: Record<Language, Dictionary> = {
  en: en as Dictionary,
  "pt-BR": ptBR as Dictionary,
};

export const LANGUAGE_LABELS: Record<Language, string> = {
  en: "English",
  "pt-BR": "Português",
};

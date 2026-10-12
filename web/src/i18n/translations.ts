import en from "../../../shared/i18n/en.json";
import ptBR from "../../../shared/i18n/pt-BR.json";

/** Supported UI languages; the dictionary lives in shared/i18n/, shared with the Android app. */
export type Language = "en" | "pt-BR";

/** Flat key → translated string map. */
export type Dictionary = Record<string, string>;

export const dictionaries: Record<Language, Dictionary> = {
  en: en as Dictionary,
  "pt-BR": ptBR as Dictionary,
};

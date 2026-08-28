import { createContext } from "react";
import type { Language } from "./translations";

export type TranslateVars = Record<string, string | number>;

export interface I18nContextValue {
  /** Currently active language ("en" | "pt-BR"). */
  lang: Language;
  setLang: (lang: Language) => void;
  /** Switches between the two supported languages. */
  toggleLang: () => void;
  /**
   * Translates a dictionary key. Supports `{name}` interpolation and simple
   * ICU-lite pluralization: `{count, plural, one {…} other {…}}`.
   */
  t: (key: string, vars?: TranslateVars) => string;
}

export const I18nContext = createContext<I18nContextValue | null>(null);

export const LANGUAGE_STORAGE_KEY = "language";

const PLURAL_RE = /\{(\w+), plural, one \{(.*?)\} other \{(.*?)\}\}/g;
const SIMPLE_RE = /\{(\w+)\}/g;

/** Replaces `{var}` placeholders and `{count, plural, one/other}` segments. */
export function interpolate(template: string, vars: TranslateVars): string {
  const withPlural = template.replace(PLURAL_RE, (_match, key, one, other) => {
    const count = Number(vars[key]);
    return count === 1 ? one : other;
  });
  return withPlural.replace(SIMPLE_RE, (match, key) => {
    const v = vars[key];
    return v === undefined || v === null ? match : String(v);
  });
}

/** Initial language: persisted choice, else browser locale (pt → pt-BR). */
export function resolveInitialLanguage(): Language {
  try {
    const saved = localStorage.getItem(LANGUAGE_STORAGE_KEY);
    if (saved === "en" || saved === "pt-BR") return saved;
  } catch {
    // Storage unavailable, fall through to the browser locale.
  }
  return navigator.language?.toLowerCase().startsWith("pt") ? "pt-BR" : "en";
}

export const MONTH_KEYS = [
  "months.jan",
  "months.feb",
  "months.mar",
  "months.apr",
  "months.may",
  "months.jun",
  "months.jul",
  "months.aug",
  "months.sep",
  "months.oct",
  "months.nov",
  "months.dec",
] as const;

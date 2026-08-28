import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { dictionaries, type Dictionary, type Language } from "./translations";
import {
  I18nContext,
  interpolate,
  LANGUAGE_STORAGE_KEY,
  resolveInitialLanguage,
  type I18nContextValue,
} from "./context";

/** Provides the active language and the `t()` translation function to the app. */
export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Language>(resolveInitialLanguage);

  const setLang = useCallback((next: Language) => {
    setLangState(next);
    try {
      localStorage.setItem(LANGUAGE_STORAGE_KEY, next);
    } catch {
      // Storage unavailable, keep the in-memory value only.
    }
  }, []);

  const toggleLang = useCallback(
    () => setLangState((prev) => (prev === "en" ? "pt-BR" : "en")),
    []
  );

  // Keep the document language attribute in sync (a11y + font rendering).
  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);

  const t = useCallback(
    (key: string, vars?: Record<string, string | number>): string => {
      const dict: Dictionary = dictionaries[lang];
      const template = dict[key] ?? dictionaries.en[key] ?? key;
      return vars ? interpolate(template, vars) : template;
    },
    [lang]
  );

  const value = useMemo<I18nContextValue>(
    () => ({ lang, setLang, toggleLang, t }),
    [lang, setLang, toggleLang, t]
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

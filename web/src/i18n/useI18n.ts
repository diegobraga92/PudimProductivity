import { useContext, useMemo } from "react";
import { I18nContext, MONTH_KEYS, type I18nContextValue } from "./context";

/** Reads the i18n context. Must be used within an `I18nProvider`. */
export function useI18n(): I18nContextValue {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used within an I18nProvider");
  }
  return ctx;
}

/** Localized month abbreviations, for use with `formatWeekRange`. */
export function useMonthNames(): string[] {
  const { t } = useI18n();
  return useMemo(() => MONTH_KEYS.map((key) => t(key)), [t]);
}

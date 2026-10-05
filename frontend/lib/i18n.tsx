"use client";

import { createContext, useContext, useEffect, useSyncExternalStore } from "react";
import { useLocalStorage } from "./useLocalStorage";
import en, { type Messages } from "./messages/en";
import ja from "./messages/ja";

export const LOCALES = ["en", "ja"] as const;
export type Locale = (typeof LOCALES)[number];

/** Shown in the language menu, each in its own language. */
export const LOCALE_NAMES: Record<Locale, string> = { en: "English", ja: "日本語" };

const MESSAGES: Record<Locale, Messages> = { en, ja };

interface I18n {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: Messages;
  /** Display name of a category: default categories are translated, custom ones shown as typed. */
  categoryName: (name: string) => string;
  formatDate: (iso: string) => string;
}

const I18nContext = createContext<I18n | null>(null);

const noSubscribe = () => () => {};

// The browser's language decides until the user picks one in the header.
function useBrowserLocale(): Locale {
  return useSyncExternalStore(
    noSubscribe,
    () => ((navigator.languages?.[0] ?? navigator.language ?? "").toLowerCase().startsWith("ja") ? "ja" : "en"),
    () => "en"
  );
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const browserLocale = useBrowserLocale();
  const [locale, setLocale] = useLocalStorage<Locale>("app.locale", browserLocale, LOCALES);
  const t = MESSAGES[locale];

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value: I18n = {
    locale,
    setLocale,
    t,
    categoryName: (name) => t.categoryNames[name.toLowerCase()] ?? name,
    formatDate: (iso) => new Date(iso).toLocaleDateString(locale === "ja" ? "ja-JP" : "en-US"),
  };

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18n {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used inside <I18nProvider>");
  return ctx;
}

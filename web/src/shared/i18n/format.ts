import type { Locale } from "./locale";

const TAG: Record<Locale, string> = { ko: "ko-KR", en: "en-US" };

export function localeTag(locale: Locale): string {
  return TAG[locale];
}

export function formatDate(date: Date, locale: Locale, options: Intl.DateTimeFormatOptions): string {
  return new Intl.DateTimeFormat(TAG[locale], options).format(date);
}

export function formatNumber(value: number, locale: Locale): string {
  return new Intl.NumberFormat(TAG[locale]).format(value);
}

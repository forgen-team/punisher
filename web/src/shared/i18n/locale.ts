import { atom, useAtomValue } from "jotai";

export const LOCALES = ["ko", "en"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "ko";

export function isLocale(value: unknown): value is Locale {
  return value === "ko" || value === "en";
}

/** 마지막으로 맞춘 화면 언어를 기억하는 localStorage 키(단독판: 첫 화면 깜박임 방지). */
export const LOCALE_STORAGE_KEY = "punisher.locale";

/**
 * 첫 화면의 언어: 지난번에 설정에서 받은 언어(localStorage). 처음이거나 저장소를 못 쓰는 환경
 * (사생활 보호 모드 등)이면 기본값 — 설정이 오면 LocaleSync 가 맞춘다.
 */
export function initialLocale(): Locale {
  try {
    const stored = window.localStorage.getItem(LOCALE_STORAGE_KEY);
    return isLocale(stored) ? stored : DEFAULT_LOCALE;
  } catch {
    return DEFAULT_LOCALE;
  }
}

/** 설정에서 온 언어를 다음 실행의 첫 화면용으로 기억한다(실패해도 화면은 그대로 동작). */
export function rememberLocale(locale: Locale): void {
  try {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale);
  } catch {
    // 저장소를 못 쓰면 다음 실행에서 설정을 받은 뒤 바뀐다 — 기억만 못 할 뿐 동작에는 지장 없음.
    return;
  }
}

const startLocale: Locale = typeof window === "undefined" ? DEFAULT_LOCALE : initialLocale();

/** 화면 언어 — 학습 설정이 오면 LocaleSync 가 맞춘다. */
export const localeAtom = atom<Locale>(startLocale);

let current: Locale = startLocale;

/** 훅 밖(에러 문구 변환 등)에서 쓰는 현재 언어 — LocaleSync 가 atom 과 함께 맞춘다. */
export function getLocale(): Locale {
  return current;
}

export function setCurrentLocale(locale: Locale): void {
  current = locale;
}

export function useLocale(): Locale {
  return useAtomValue(localeAtom);
}

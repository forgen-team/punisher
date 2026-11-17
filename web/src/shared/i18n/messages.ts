import { getLocale, useLocale, type Locale } from "./locale";

type Template = string | ((...args: never[]) => string);
/** 한 모듈의 카탈로그: 영어는 한국어와 같은 키·같은 모양이어야 한다(타입으로 강제). */
export type Messages<K extends Record<string, Template>> = { ko: K; en: { [P in keyof K]: K[P] } };

export function defineMessages<K extends Record<string, Template>>(messages: Messages<K>): Messages<K> {
  return messages;
}

export function translate<K extends Record<string, Template>>(messages: Messages<K>, locale: Locale = getLocale()): K {
  return (messages[locale] ?? messages.ko) as K;
}

/** 현재 언어의 카탈로그 — `const t = useT(messages); t.title; t.count(3)` */
export function useT<K extends Record<string, Template>>(messages: Messages<K>): K {
  return translate(messages, useLocale());
}

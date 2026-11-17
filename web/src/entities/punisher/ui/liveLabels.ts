import { getLocale, type Locale } from "@/src/shared/i18n";

/**
 * 언어별 이름표 표를 같은 모양의 Record 로 노출한다 — 값을 읽을 때마다 현재 언어(getLocale)를 따른다.
 * 그래서 기존 `X_LABELS[key]` 호출처는 그대로 둬도 된다. 렌더 중 언어 변화를 확실히 반영하려면
 * `X_LABELS_BY_LOCALE[useLocale()][key]` 를 쓴다.
 */
export function liveLabels<K extends PropertyKey>(byLocale: Record<Locale, Record<K, string>>): Record<K, string> {
  const view = {} as Record<K, string>;
  for (const key of Object.keys(byLocale.ko) as K[]) {
    Object.defineProperty(view, key, { enumerable: true, get: () => byLocale[getLocale()][key] ?? byLocale.ko[key] });
  }
  return view;
}

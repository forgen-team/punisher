/**
 * Punisher 화면 문구(한국어·영어). 라이브러리 없이 모듈마다 `defineMessages({ ko, en })` 로 카탈로그를 두고
 * 컴포넌트는 `useT(messages)`, 훅 밖은 `translate(messages)` 로 읽는다. 언어는 학습 설정(`locale`)에서 오고
 * (`LocaleSync`), 기본은 한국어. 날짜·숫자는 `formatDate`/`formatNumber`(locale 고정 문자열 금지).
 */
export {
  localeAtom,
  useLocale,
  getLocale,
  setCurrentLocale,
  DEFAULT_LOCALE,
  LOCALES,
  isLocale,
  initialLocale,
  rememberLocale,
  LOCALE_STORAGE_KEY,
  type Locale,
} from "./locale";
export { defineMessages, useT, translate, type Messages } from "./messages";
export { formatDate, formatNumber, localeTag } from "./format";

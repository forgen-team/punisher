import { getLocale, translate, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

/**
 * 끝까지 읽었지만 '완료'를 누르지 않은 강의 — 서버는 계속 진행 중(completed_at 없음)으로 두고,
 * 화면에서만 "100%"·"남은 시간" 대신 '다 읽음'으로 보인다.
 */
export function isReadThrough(percent: number | undefined, completed: boolean): boolean {
  return !completed && (percent ?? 0) >= 100;
}

/** 현재 언어의 '다 읽음' 문구 — 렌더 중이면 `readThroughLabel(useLocale())`. */
export function readThroughLabel(locale: Locale = getLocale()): string {
  return translate(messages, locale).readThrough;
}

/** @deprecated 고정 한국어 — 언어를 따르려면 readThroughLabel(locale). 기존 호출처 호환용. */
export const READ_THROUGH_LABEL = translate(messages, "ko").readThrough;

import type { TodaySession } from "@/src/shared/api/punisher";
import { translate, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

/**
 * 오늘 학습 세션 시간(v2.2 §5) — 무엇의 합인지 보이게 "오늘 할 일 약 29분: 강의 20 · 확인 7 · 복습 2".
 * 항목은 각각 반올림하고 합계는 그 합(보이는 숫자끼리 맞는다). 0분인 항목은 뺀다(복습 만기가 없는 날 등).
 * 모두 0이거나 응답에 없으면 빈 문자열.
 */
export function sessionTimeLabel(session: TodaySession | undefined, locale: Locale): string {
  if (!session) { return ""; }
  const t = translate(messages, locale);
  const parts: Array<[(minutes: number) => string, number]> = [
    [t.sessionLecture, session.lectureMinutes],
    [t.sessionCheck, session.checkMinutes],
    [t.sessionReview, session.reviewMinutes],
  ];
  const shown = parts
    .filter(([, minutes]) => Number.isFinite(minutes) && minutes > 0)
    .map(([label, minutes]) => [label, Math.max(1, Math.round(minutes))] as const);
  if (shown.length === 0) { return ""; }
  const total = shown.reduce((sum, [, minutes]) => sum + minutes, 0);
  return t.sessionTotal(total, shown.map(([label, minutes]) => label(minutes)).join(" · "));
}

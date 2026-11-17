import type { TodaySession } from "@/src/shared/api/punisher";
import { translate, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

/**
 * 오늘 학습 세션 시간 합계(v2.2 §5) — "강의 약 n분 + 확인 약 m분 + 복습 약 k분".
 * 0분인 항목은 뺀다(복습 만기가 없는 날 등). 모두 0이거나 응답에 없으면 빈 문자열.
 */
export function sessionTimeLabel(session: TodaySession | undefined, locale: Locale): string {
  if (!session) { return ""; }
  const t = translate(messages, locale);
  const parts: Array<[(minutes: number) => string, number]> = [
    [t.sessionLecture, session.lectureMinutes],
    [t.sessionCheck, session.checkMinutes],
    [t.sessionReview, session.reviewMinutes],
  ];
  return parts
    .filter(([, minutes]) => Number.isFinite(minutes) && minutes > 0)
    .map(([label, minutes]) => label(Math.round(minutes)))
    .join(" + ");
}

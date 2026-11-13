import type { TodaySession } from "@/src/shared/api/punisher";

/**
 * 오늘 학습 세션 시간 합계(v2.2 §5) — "강의 약 n분 + 확인 약 m분 + 복습 약 k분".
 * 0분인 항목은 뺀다(복습 만기가 없는 날 등). 모두 0이거나 응답에 없으면 빈 문자열.
 */
export function sessionTimeLabel(session: TodaySession | undefined): string {
  if (!session) { return ""; }
  const parts: Array<[string, number]> = [
    ["강의", session.lectureMinutes],
    ["확인", session.checkMinutes],
    ["복습", session.reviewMinutes],
  ];
  return parts
    .filter(([, minutes]) => Number.isFinite(minutes) && minutes > 0)
    .map(([label, minutes]) => `${label} 약 ${Math.round(minutes)}분`)
    .join(" + ");
}

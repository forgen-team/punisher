import { REVIEW_INTERVALS_MAX, REVIEW_MINUTES_MAX, type LearningSettingsPatch } from "@/src/shared/api/punisher";

export type SettingsField = keyof LearningSettingsPatch;

/** 숙달 기준으로 고를 수 있는 레벨(서버 learning.ValidMasteryTarget). */
export const MASTERY_TARGET_OPTIONS = [3, 4, 5] as const;

/** 하루 복습 시간 선택지(0~30분). */
export const REVIEW_MINUTE_OPTIONS = Array.from({ length: REVIEW_MINUTES_MAX / 5 + 1 }, (_, index) => index * 5);

/** 복습 간격 입력("1, 7, 30") → 일 수 목록. 양의 정수 1~6개이고 증가해야 한다. */
export function parseIntervals(text: string): { days: number[] } | { error: string } {
  const parts = text.split(/[\s,]+/).filter(Boolean);
  const days = parts.map(Number);
  if (days.length === 0 || days.length > REVIEW_INTERVALS_MAX) {
    return { error: `간격은 1~${REVIEW_INTERVALS_MAX}개로 적어 주세요.` };
  }
  if (days.some((d) => !Number.isInteger(d) || d <= 0)) { return { error: "간격은 1 이상의 정수(일)로 적어 주세요." }; }
  if (days.some((d, index) => index > 0 && d <= days[index - 1])) { return { error: "간격은 점점 커져야 해요." }; }
  return { days };
}

export function formatIntervals(days: number[]): string {
  return days.join(", ");
}

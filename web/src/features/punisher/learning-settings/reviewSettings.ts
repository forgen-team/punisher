import { REVIEW_INTERVALS_MAX, REVIEW_MINUTES_MAX, type LearningSettingsPatch } from "@/src/shared/api/punisher";
import { translate } from "@/src/shared/i18n";
import { messages } from "./i18n";

export type SettingsField = keyof LearningSettingsPatch;

/** 숙달 기준으로 고를 수 있는 레벨(서버 learning.ValidMasteryTarget). */
export const MASTERY_TARGET_OPTIONS = [3, 4, 5] as const;

/** 하루 복습 시간 선택지(0~30분). */
export const REVIEW_MINUTE_OPTIONS = Array.from({ length: REVIEW_MINUTES_MAX / 5 + 1 }, (_, index) => index * 5);

/** 복습 간격 입력("1, 7, 30") → 일 수 목록. 양의 정수 1~6개이고 증가해야 한다. */
export function parseIntervals(text: string): { days: number[] } | { error: string } {
  const parts = text.split(/[\s,]+/).filter(Boolean);
  const days = parts.map(Number);
  const t = translate(messages);
  if (days.length === 0 || days.length > REVIEW_INTERVALS_MAX) {
    return { error: t.intervalsCount(REVIEW_INTERVALS_MAX) };
  }
  if (days.some((d) => !Number.isInteger(d) || d <= 0)) { return { error: t.intervalsInteger }; }
  if (days.some((d, index) => index > 0 && d <= days[index - 1])) { return { error: t.intervalsIncreasing }; }
  return { days };
}

export function formatIntervals(days: number[]): string {
  return days.join(", ");
}

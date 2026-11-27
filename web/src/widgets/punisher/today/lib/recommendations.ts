import { lectureHref, startLectureHref } from "@/src/entities/punisher";
import type { NextChapterPrompt, Recommendation, TodaySession } from "@/src/shared/api/punisher";

export const NEW_CHAPTER_PATH = "/chapters/new";

/** 추천 행 = 강의 열기(D13): 강의가 있으면 그 강의, 없으면 /start 로 만들기까지(진짜 링크). */
export function recommendationHref(item: Pick<Recommendation, "conceptId" | "lectureId">): string {
  return item.lectureId ? lectureHref(item.lectureId) : startLectureHref(item.conceptId);
}

/** 지도에서 그 개념을 연 주소. */
export function mapNodeHref(conceptId: string): string {
  return `/map?node=${encodeURIComponent(conceptId)}`;
}

/** 오늘의 추천 머리의 "강의 약 n분" — 서버 세션 합계의 강의 몫, 없으면 추천 분량 합. 0 이면 0. */
export function lectureMinutes(session: TodaySession | undefined, items: readonly Pick<Recommendation, "estimatedMinutes">[]): number {
  const fromSession = session?.lectureMinutes ?? 0;
  const minutes = fromSession > 0 ? fromSession : items.reduce((sum, item) => sum + (item.estimatedMinutes || 0), 0);
  return Number.isFinite(minutes) && minutes > 0 ? Math.max(1, Math.round(minutes)) : 0;
}

/** 끝 줄 "남은 강의 약 n일 치" 의 n — 서버가 nextChapter 를 줄 때만(남은 분량 < 하루 학습량 × 14일). */
export function chapterDaysLeft(prompt: NextChapterPrompt): number {
  return prompt.dailyMinutes > 0 ? Math.max(0, Math.floor(prompt.remainingMinutes / prompt.dailyMinutes)) : 0;
}

/** 이어보기·추천·공부할 주제 묶음에서 먼저 보이는 행 수 — 넘으면 "n개 더". */
export const BUNDLE_LIMIT = 3;

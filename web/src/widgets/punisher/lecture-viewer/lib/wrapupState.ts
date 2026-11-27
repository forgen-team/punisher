import type { CheckHistoryItem, ConceptMastery, Recommendation } from "@/src/shared/api/punisher";

/** [다음 강의: 제목] 대상 — 오늘의 추천 첫 항목(이어보기·이 강의·같은 개념 제외). */
export interface NextLectureTarget {
  title: string;
  lectureId?: string;
  conceptId: string;
}

export function pickNextLecture(items: Recommendation[] | undefined, lectureId: string, conceptId: string): NextLectureTarget | null {
  const next = (items ?? []).find(
    (r) => r.kind !== "resume" && r.lectureId !== lectureId && r.conceptId !== conceptId,
  );
  if (!next) { return null; }
  return { title: next.conceptName || next.title, lectureId: next.lectureId, conceptId: next.conceptId };
}

/** 이 강의의 가장 최근 끝난 확인하기(목록은 최신 순). */
export function lastFinishedCheck(sessions: CheckHistoryItem[] | undefined, lectureId: string): CheckHistoryItem | null {
  return (sessions ?? []).find((s) => s.kind === "check" && s.lectureId === lectureId && s.status === "finished" && s.itemCount > 0) ?? null;
}

/** 이 강의 목표 중 가장 이른 다음 복습일(없으면 null). */
export function earliestReview(mastery: ConceptMastery | undefined, objectiveIds: string[]): Date | null {
  const ids = new Set(objectiveIds);
  let best: Date | null = null;
  for (const o of mastery?.objectives ?? []) {
    if (!o.nextReviewAt || (ids.size > 0 && !ids.has(o.objectiveId))) { continue; }
    const at = new Date(o.nextReviewAt);
    if (Number.isNaN(at.getTime())) { continue; }
    if (!best || at < best) { best = at; }
  }
  return best;
}

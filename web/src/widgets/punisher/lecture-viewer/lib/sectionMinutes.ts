import type { LectureSection } from "@/src/shared/api/punisher";

/** 섹션에 보일 분 — 서버 계산값(estimatedMinutes)을 우선, 없으면(생성 중 미리보기·옛 응답) 모델이 쓴 minutes. */
export function sectionMinutes(section: Pick<LectureSection, "minutes" | "estimatedMinutes">): number {
  return section.estimatedMinutes ?? section.minutes;
}

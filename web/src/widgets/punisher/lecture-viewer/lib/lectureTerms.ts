import type { LectureContent } from "@/src/shared/api/punisher";
import { toInlineTermLinks } from "./inlineTerms";

/** 본문에 나온 용어 하나 — 처음 나온 섹션 위치와 함께. */
export interface LectureTerm {
  conceptId: string;
  term: string;
  /** 처음 나온 섹션 anchor */
  sectionAnchor: string;
}

const INLINE_TERM = /\[([^\]]+)\]\(term:([\w.-]+)\)/g;

/**
 * 강의에 나온 용어 목록(보조 열 '용어' 탭) — 용어 블록과 문장 안 [용어](term:id) 링크를 나온 순서대로,
 * 같은 개념은 한 번만.
 */
export function extractLectureTerms(content: Pick<LectureContent, "sections">): LectureTerm[] {
  const seen = new Map<string, LectureTerm>();
  const add = (conceptId: string, term: string, sectionAnchor: string) => {
    if (!conceptId || seen.has(conceptId)) { return; }
    seen.set(conceptId, { conceptId, term: term.trim() || conceptId, sectionAnchor });
  };
  for (const section of content.sections) {
    for (const block of section.blocks) {
      if (block.type === "term" && block.conceptId) {
        add(block.conceptId, block.term ?? "", section.anchor);
        continue;
      }
      if (block.type !== "markdown" && block.type !== "misconception") { continue; }
      for (const match of toInlineTermLinks(block.text ?? "").matchAll(INLINE_TERM)) {
        add(match[2], match[1], section.anchor);
      }
    }
  }
  return [...seen.values()];
}

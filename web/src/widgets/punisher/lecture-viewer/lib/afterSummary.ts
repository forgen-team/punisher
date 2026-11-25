/**
 * judgment_next 에서 '다음에 배울 개념'(지식 지도 링크)을 요약 뒤로 — 요약을 먼저 떠올린 뒤 보이게(조사 #9).
 * 새 강의는 다음 개념을 @@summary 뒤 @@markdown 에 쓴다. 그 전 강의는 요약 바로 앞 마크다운의
 * "다음(으로|에) 배울 …" 문단부터 끝까지를 떼어 요약 뒤로 옮긴다(나머지 글은 그 자리에 남는다).
 */
import type { LectureBlock } from "@/src/shared/api/punisher";

export interface SummaryLayout {
  before: LectureBlock[];
  summary: LectureBlock;
  /** 요약을 떠올린 뒤에 보일 블록 */
  after: LectureBlock[];
}

const NEXT_HEADING = /^(?:#{1,6}\s*)?(?:\*\*)?(?:다음(?:으로|에)?\s*(?:배울|볼|공부할|읽을)|다음\s*개념|what to learn next|next to learn|next concepts?|where to go next|up next)/i;

/** 요약이 없으면 null(그대로 그린다). */
export function layoutAroundSummary(blocks: LectureBlock[]): SummaryLayout | null {
  const at = blocks.findIndex((b) => b.type === "summary");
  if (at < 0) { return null; }
  const before = blocks.slice(0, at);
  const after = blocks.slice(at + 1);
  const last = before[before.length - 1];
  if (last?.type === "markdown" && last.text) {
    const paragraphs = last.text.split(/\n{2,}/);
    const cut = paragraphs.findIndex((p) => NEXT_HEADING.test(p.trim()));
    if (cut === 0) {
      return { before: before.slice(0, -1), summary: blocks[at], after: [last, ...after] };
    }
    if (cut > 0) {
      const kept: LectureBlock = { ...last, text: paragraphs.slice(0, cut).join("\n\n") };
      const moved: LectureBlock = { anchor: "", type: "markdown", text: paragraphs.slice(cut).join("\n\n") };
      return { before: [...before.slice(0, -1), kept], summary: blocks[at], after: [moved, ...after] };
    }
  }
  return { before, summary: blocks[at], after };
}

import { lectureAnchorHref } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import type { SearchHit } from "@/src/shared/api/punisher";

/** 결과로 가는 주소: 강의(그 섹션)·메모·질문은 강의 안 자리, 질문 해설은 그 화면. */
export function hitHref(hit: SearchHit): string {
  if (hit.kind === "inquiry") { return inquiryHref(hit.refId); }
  return hit.lectureId ? lectureAnchorHref(hit.lectureId, hit.anchor) : "/history?tab=notes";
}

export interface SnippetPart {
  text: string;
  match: boolean;
}

/** 서버 스니펫의 [[ ]] 표지를 강조 조각으로 나눈다. 표지가 없으면(짧은 검색어) q 를 찾아 강조한다. */
export function snippetParts(snippet: string, q: string): SnippetPart[] {
  if (snippet.includes("[[")) {
    return snippet.split(/(\[\[.*?\]\])/).filter(Boolean).map((part) =>
      part.startsWith("[[") && part.endsWith("]]") ? { text: part.slice(2, -2), match: true } : { text: part, match: false });
  }
  const needle = q.trim();
  if (!needle) { return [{ text: snippet, match: false }]; }
  const out: SnippetPart[] = [];
  let rest = snippet;
  for (let at = rest.indexOf(needle); at >= 0; at = rest.indexOf(needle)) {
    if (at > 0) { out.push({ text: rest.slice(0, at), match: false }); }
    out.push({ text: needle, match: true });
    rest = rest.slice(at + needle.length);
  }
  if (rest) { out.push({ text: rest, match: false }); }
  return out;
}

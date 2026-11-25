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

/** 결과 묶음: 같은 강의의 섹션 결과는 한 묶음(강의 제목 1줄 + 섹션 여러 줄), 그 밖은 한 줄씩. 처음 나온 순서를 지킨다. */
export type HitGroup =
  | { type: "lecture"; lectureId: string; title: string; hits: SearchHit[] }
  | { type: "single"; hit: SearchHit };

export function groupHits(hits: SearchHit[]): HitGroup[] {
  const out: HitGroup[] = [];
  const byLecture = new Map<string, Extract<HitGroup, { type: "lecture" }>>();
  for (const hit of hits) {
    if (hit.kind !== "lecture" || !hit.lectureId) {
      out.push({ type: "single", hit });
      continue;
    }
    const group = byLecture.get(hit.lectureId);
    if (group) {
      group.hits.push(hit);
      continue;
    }
    const next = { type: "lecture" as const, lectureId: hit.lectureId, title: hit.title, hits: [hit] };
    byLecture.set(hit.lectureId, next);
    out.push(next);
  }
  return out;
}

/** "강의 n편 · m곳 (· 그 밖에 k개)" 의 숫자. */
export function hitCounts(groups: HitGroup[]): { lectures: number; places: number; others: number } {
  let lectures = 0;
  let places = 0;
  let others = 0;
  for (const g of groups) {
    if (g.type === "lecture") {
      lectures += 1;
      places += g.hits.length;
    } else {
      others += 1;
    }
  }
  return { lectures, places, others };
}

import type { DocumentLink, LectureContent } from "@/src/shared/api/punisher";

/** 강의 끝 '이 강의의 출처' — 섹션 출처 블록에 이미 나온 링크(URL)는 다시 나열하지 않는다. */
export function sourcesNotInSections(sources: readonly DocumentLink[], content: Pick<LectureContent, "sections">): DocumentLink[] {
  const shown = new Set(content.sections.flatMap((s) => s.blocks.flatMap((b) => (b.type === "sources" ? (b.sources ?? []).map((l) => l.url) : []))));
  return sources.filter((s) => !shown.has(s.url));
}

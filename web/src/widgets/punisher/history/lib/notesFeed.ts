import { lectureAnchorHref } from "@/src/entities/punisher";
import { inquiryHref } from "@/src/features/punisher/instant-start";
import type { Concept, Inquiry, MyQuestion, Note, SearchHit } from "@/src/shared/api/punisher";
import type { NoteKind } from "./historyParams";

/** 질문·메모 탭의 한 줄. href 가 없으면 눌러도 갈 곳이 없는 메모(강의에 붙지 않은 것). */
export interface NoteRow {
  key: string;
  kind: NoteKind;
  title: string;
  /** 종류 이름 뒤에 붙는 조각(강의 이름·날짜 …) — 호출한 쪽이 " · " 로 잇는다. */
  parts: string[];
  href?: string;
  /** 정렬용 시각(ISO). 검색 결과에는 없다. */
  at: string;
}

/** 강의 안 자리 + 옆 열(질문·메모) 열기. */
export function lectureNotesHref(lectureId: string, anchor: string | null | undefined): string {
  const base = lectureAnchorHref(lectureId, anchor);
  return `${base}${base.includes("?") ? "&" : "?"}panel=notes`;
}

export function inquiryRow(i: Inquiry, title: string, status: string, day: string): NoteRow {
  return { key: `inquiry:${i.id}`, kind: "inquiry", title, parts: [status, day], href: inquiryHref(i.id), at: i.createdAt };
}

export function questionRow(q: MyQuestion, day: string): NoteRow {
  return {
    key: `lq:${q.id}`,
    kind: "lq",
    title: q.question,
    parts: [q.lectureTitle || q.conceptName, day],
    href: q.lectureId ? lectureNotesHref(q.lectureId, q.anchor) : undefined,
    at: q.createdAt,
  };
}

export function memoRow(n: Note, lectureName: string, day: string): NoteRow {
  return {
    key: `memo:${n.id}`,
    kind: "memo",
    title: n.body || n.quote,
    parts: [lectureName, day],
    href: n.lectureId ? lectureNotesHref(n.lectureId, n.anchor) : undefined,
    at: n.createdAt,
  };
}

/** 날짜 최신순(같으면 들어온 순서). */
export function sortNoteRows(rows: NoteRow[]): NoteRow[] {
  return rows
    .map((row, index) => ({ row, index }))
    .sort((a, b) => b.row.at.localeCompare(a.row.at) || a.index - b.index)
    .map(({ row }) => row);
}

export function filterByKind<T extends { kind: NoteKind }>(rows: T[], kind: NoteKind | ""): T[] {
  return kind ? rows.filter((row) => row.kind === kind) : rows;
}

/** 검색 결과의 종류 — 강의 본문은 학습 탭 몫이라 null. */
export function hitNoteKind(hit: SearchHit): NoteKind | null {
  if (hit.kind === "inquiry") { return "inquiry"; }
  if (hit.kind === "question") { return "lq"; }
  if (hit.kind === "note") { return "memo"; }
  return null;
}

/** 검색 결과 중 질문·메모 탭 몫(강의 본문 제외), 종류가 붙은 채로. */
export function noteHits(hits: SearchHit[]): Array<{ hit: SearchHit; kind: NoteKind }> {
  return hits.flatMap((hit) => {
    const kind = hitNoteKind(hit);
    return kind ? [{ hit, kind }] : [];
  });
}

export function noteHitHref(hit: SearchHit, kind: NoteKind): string | undefined {
  if (kind === "inquiry") { return inquiryHref(hit.refId); }
  return hit.lectureId ? lectureNotesHref(hit.lectureId, hit.anchor) : undefined;
}

/** 완료·이미 앎 개념 중 검색어(이름)와 영역에 맞는 것, 최근 갱신순. */
export function matchLearnedConcepts(concepts: Concept[], q: string, area: string): Concept[] {
  const needle = q.trim().toLowerCase();
  return concepts
    .filter((c) => (c.userState === "completed" || c.userState === "known") && (!area || c.area === area) && (!needle || c.name.toLowerCase().includes(needle)))
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
}

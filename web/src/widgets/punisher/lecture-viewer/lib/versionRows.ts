import type { LinkedLecture } from "@/src/shared/api/punisher";

/** 지금 보는 강의(판 목록 맨 위에 '지금 보는 판'으로 같이 보인다). */
export interface CurrentVersion {
  id: string;
  title: string;
  version: number;
  createdAt: string;
  promptVersion: string;
}

export interface VersionRow {
  id: string;
  version: number;
  createdAt: string;
  promptVersion: string;
  /** 지금 보는 판과 제목이 다를 때만 — 같으면 "" */
  distinctTitle: string;
  current: boolean;
  linked?: LinkedLecture;
}

/**
 * '새 프롬프트로 다시 만든 강의' 줄: 같은 제목이 반복돼 구분이 안 되던 것을 판 번호·만든 날·프롬프트 버전으로 가른다.
 * 지금 보는 판을 함께 넣고 판 번호 순(같으면 만든 날 순)으로 둔다.
 */
export function versionRows(linked: LinkedLecture[], current: CurrentVersion | undefined): VersionRow[] {
  const base = current?.title.trim() ?? "";
  const rows: VersionRow[] = linked.map((l) => ({
    id: l.id, version: l.version, createdAt: l.createdAt, promptVersion: l.promptVersion,
    distinctTitle: l.title.trim() && l.title.trim() !== base ? l.title.trim() : "", current: false, linked: l,
  }));
  if (current && !rows.some((r) => r.id === current.id)) {
    rows.push({ id: current.id, version: current.version, createdAt: current.createdAt, promptVersion: current.promptVersion, distinctTitle: "", current: true });
  }
  return rows.sort((a, b) => a.version - b.version || a.createdAt.localeCompare(b.createdAt));
}

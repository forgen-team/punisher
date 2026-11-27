export type HistoryTab = "learn" | "review" | "notes";
export type NoteKind = "inquiry" | "lq" | "memo";

export const HISTORY_TABS: HistoryTab[] = ["learn", "review", "notes"];
export const NOTE_KINDS: NoteKind[] = ["inquiry", "lq", "memo"];

/** 옛 6탭 값 → 새 탭(질문 해설·강의 중 질문·메모는 질문·메모 탭의 종류 칩). `notes` 는 새 탭 이름이라 그대로 새 뜻이다. */
const LEGACY_TABS: Record<string, { tab: HistoryTab; kind?: NoteKind }> = {
  lectures: { tab: "learn" },
  learned: { tab: "learn" },
  checks: { tab: "review" },
  inquiries: { tab: "notes", kind: "inquiry" },
  questions: { tab: "notes", kind: "lq" },
};

export interface HistoryView {
  tab: HistoryTab;
  /** 질문·메모 탭의 종류(없으면 모두) */
  kind: NoteKind | "";
  /** 주소가 옛 값이라 새 주소로 바꿔야 한다 */
  legacy: boolean;
}

/** 주소의 ?tab=·?kind= 를 화면 상태로. 모르는 값은 기본(학습)·모두. */
export function resolveHistoryView(tabParam: string | null, kindParam: string | null): HistoryView {
  const legacy = tabParam ? LEGACY_TABS[tabParam] : undefined;
  if (legacy) { return { tab: legacy.tab, kind: legacy.kind ?? "", legacy: true }; }
  const tab = HISTORY_TABS.find((id) => id === tabParam) ?? "learn";
  const kind = NOTE_KINDS.find((id) => id === kindParam) ?? "";
  return { tab, kind: tab === "notes" ? kind : "", legacy: false };
}

export interface HistoryLocation {
  tab: HistoryTab;
  q?: string;
  area?: string;
  kind?: NoteKind | "";
}

/** 기록 주소. 검색어는 탭을 건너 가져가고, 영역은 학습 탭에서만·종류는 질문·메모 탭에서만 남는다. */
export function historyHref(pathname: string, loc: HistoryLocation): string {
  const next = new URLSearchParams({ tab: loc.tab });
  if (loc.q) { next.set("q", loc.q); }
  if (loc.tab === "learn" && loc.area) { next.set("area", loc.area); }
  if (loc.tab === "notes" && loc.kind) { next.set("kind", loc.kind); }
  return `${pathname}?${next.toString()}`;
}

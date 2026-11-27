/**
 * 섹션 [오늘·지도·기록] — 경로가 어느 섹션에 속하는지(manifest nav.items 의 href·match 와 같다, 재설계 §1-1)와
 * 섹션마다 마지막 주소·스크롤 기억(policy/20 §4: 돌아오면 복원, 지금 섹션을 다시 누르면 첫 화면).
 */
export type LearnSegment = "today" | "map" | "history";

export const HREFS: Record<LearnSegment, string> = {
  today: "/",
  map: "/map",
  history: "/history",
};
export const SEGMENTS = ["today", "map", "history"] as const;

/** 섹션 첫 화면이 아닌데도 그 섹션에 속하는 경로 앞부분(internal/witems/app.go nav.items match 와 같게). */
const MATCH: Record<LearnSegment, string[]> = {
  today: ["/lectures", "/check", "/cards", "/inquiries", "/instant", "/start"],
  map: ["/chapters"],
  history: [],
};

function under(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

export function segmentOf(pathname: string): LearnSegment | null {
  if (pathname === "/" || pathname === "") { return "today"; }
  for (const segment of SEGMENTS) {
    if (segment !== "today" && under(pathname, HREFS[segment])) { return segment; }
    if (MATCH[segment].some((prefix) => under(pathname, prefix))) { return segment; }
  }
  return null;
}

/** 섹션 첫 화면(← 를 그리지 않는 곳). */
export function isSectionRoot(pathname: string): boolean {
  return SEGMENTS.some((segment) => HREFS[segment] === (pathname || "/"));
}

interface TabEntry {
  href: string;
  scrollY?: number;
}

const memory = new Map<LearnSegment, TabEntry>();

/** 섹션의 지금 주소(scrollY 를 주면 그 자리도). 주소가 바뀌면 옛 스크롤은 버린다. */
export function rememberTab(segment: LearnSegment, href: string, scrollY?: number) {
  const prev = memory.get(segment);
  if (scrollY !== undefined) {
    memory.set(segment, { href, scrollY });
    return;
  }
  memory.set(segment, prev && prev.href === href ? prev : { href });
}

/** 다른 섹션으로 갈 때의 주소 — 그 섹션에서 마지막으로 본 주소, 없으면 첫 화면. */
export function tabHref(segment: LearnSegment): string {
  const entry = memory.get(segment);
  return entry && segmentOf(entry.href.split("?")[0] ?? "") === segment ? entry.href : HREFS[segment];
}

export function tabScroll(segment: LearnSegment): number | undefined {
  return memory.get(segment)?.scrollY;
}

/** 테스트용 */
export function resetTabMemory() {
  memory.clear();
}

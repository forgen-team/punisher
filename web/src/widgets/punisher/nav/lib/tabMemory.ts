/** 학습 탭 — 탭마다 마지막 주소·스크롤을 기억한다(UI-PAGE-LAYOUT §2: 돌아오면 복원, 지금 탭 다시 누르면 처음). */
export type LearnSegment = "lectures" | "map" | "history";

export const HREFS: Record<LearnSegment, string> = {
  lectures: "/",
  map: "/map",
  history: "/history",
};
export const SEGMENTS = ["lectures", "map", "history"] as const;

export function segmentOf(pathname: string): LearnSegment | null {
  if (pathname === "/" || pathname === "") { return "lectures"; }
  if (pathname.startsWith("/map") || pathname.startsWith("/goals")) { return "map"; }
  if (pathname.startsWith("/history")) { return "history"; }
  return null;
}

interface TabEntry {
  href: string;
  scrollY?: number;
}

const memory = new Map<LearnSegment, TabEntry>();

/** 탭의 지금 주소(scrollY 를 주면 그 자리도). 주소가 바뀌면 옛 스크롤은 버린다. */
export function rememberTab(segment: LearnSegment, href: string, scrollY?: number) {
  const prev = memory.get(segment);
  if (scrollY !== undefined) {
    memory.set(segment, { href, scrollY });
    return;
  }
  memory.set(segment, prev && prev.href === href ? prev : { href });
}

/** 다른 탭으로 갈 때의 주소 — 그 탭에서 마지막으로 본 주소, 없으면 처음 화면. */
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

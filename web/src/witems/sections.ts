import { HREFS, segmentOf, type LearnSegment } from "@/src/widgets/punisher/nav";

/**
 * Witems 셸 사이드바 하위 메뉴(manifest nav.items — internal/witems/app.go)와 앱 경로를 잇는다.
 * 셸은 하위 메뉴를 해시(#/map)로 그리고 지금 해시로 현재 항목을 고르므로, 셸 주소를 "경로#섹션"
 * (/map?concept=a → /map?concept=a#/map)으로 둔다 — 경로는 onRouteChange·새로고침에, 해시는 셸 표시에 쓴다.
 */

/** 하위 메뉴 항목의 해시(nav.items href) — 셸이 이 해시로 그 탭을 연다. */
export const SECTION_HASH: Record<LearnSegment, string> = {
  lectures: "#/",
  map: "#/map",
  history: "#/history",
};

function under(pathname: string, base: string): boolean {
  return pathname === base || pathname.startsWith(`${base}/`);
}

/** 앱 경로의 셸 해시. 탭 밖 화면(설정·검색 등)은 null — 셸 주소에 해시를 붙이지 않는다. */
export function sectionHashOf(pathname: string): string | null {
  if (under(pathname, "/lectures")) { return "#/lectures"; }
  if (under(pathname, "/goals")) { return "#/goals"; }
  const segment = segmentOf(pathname);
  return segment ? SECTION_HASH[segment] : null;
}

/** 앱 경로 → 셸에 넘길 경로("/history?area=net" → "/history?area=net#/history"). */
export function toShellPath(to: string): string {
  const [pathname = "/"] = to.split(/[?#]/, 1);
  const hash = sectionHashOf(pathname || "/");
  return hash ? `${to}#${hash.slice(1)}` : to;
}

/** 셸 해시 → 그 하위 메뉴의 탭. 하위 메뉴 해시가 아니면 null(#/goals 는 지도). */
export function segmentOfHash(hash: string): LearnSegment | null {
  if (hash === "#/goals") { return "map"; }
  const found = (Object.keys(SECTION_HASH) as LearnSegment[]).find((segment) => SECTION_HASH[segment] === hash);
  return found ?? null;
}

/**
 * 셸이 알린 주소(경로 + 지금 해시)에서 앱이 갈 탭. 해시가 경로와 맞으면(뒤로가기·새로고침) null = 경로 그대로.
 * 해시가 다른 탭이면 셸 하위 메뉴를 누른 것이다 — 다른 하위 화면에서 누르면 셸은 /apps/punisher#/map 으로
 * 옮겨 hashchange 없이 경로 "/" 만 알린다.
 */
export function segmentFromShell(path: string, hash: string): LearnSegment | null {
  const segment = segmentOfHash(hash);
  if (!segment) { return null; }
  const [pathname = "/"] = path.split(/[?#]/, 1);
  return sectionHashOf(pathname || "/") === hash ? null : segment;
}

/** mount 때 처음 볼 앱 경로: 셸 하위 메뉴로 들어오면 그 탭의 처음 화면, 아니면 ctx.path. */
export function initialAppPath(path: string, search: string, hash: string): string {
  const segment = segmentFromShell(path, hash);
  return segment ? HREFS[segment] : `${path}${search}`;
}

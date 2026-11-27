/**
 * 옛 해시 주소 호환(D8 이전, 한 릴리스용 — 다음 릴리스에서 지운다). 예전 manifest nav.items 는 "#/map" 같은
 * 해시였고, 앱은 셸 주소를 "경로#섹션"(/map?concept=a#/map, /lectures/l1#/lectures)으로 두었다.
 * 그런 주소로 들어오면 해시를 뺀 경로를 돌려준다(mount 가 ctx.navigate(…, { replace: true }) 로 바로잡는다).
 * - 경로가 이미 있으면(/map#/map) 경로·쿼리 그대로, 해시만 뺀다.
 * - basePath 에 해시만 있으면(/apps/punisher#/history, #/?today) 해시 안의 경로·쿼리로 간다. 섹션이 아니면 basePath.
 * 해시가 "#/" 로 시작하지 않으면(본문 앵커 등) null — 손대지 않는다.
 */
const SECTIONS = new Set(["/", "/map", "/goals", "/history"]);

export function legacyHashPath(path: string, search: string, hash: string): string | null {
  if (!hash.startsWith("#/")) { return null; }
  if (path !== "/") { return `${path}${search}`; }
  const body = hash.slice(1);
  const [pathname = "/"] = body.split("?", 1);
  if (pathname === "/lectures" || pathname.startsWith("/lectures/")) { return `/${body.slice(pathname.length)}`; }
  return SECTIONS.has(pathname) ? body : `/${search}`;
}

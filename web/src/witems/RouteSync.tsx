import { useEffect, useRef } from "react";
import { useLocation, useNavigate, useNavigationType } from "react-router-dom";
import { SHELL_NAV_STATE, isShellNavigation } from "@/src/shared/lib/in-app-history";
import { HREFS, rememberTab, segmentOf, tabHref, tabScroll, type LearnSegment } from "@/src/widgets/punisher/nav";
import type { AppContext } from "./contract";
import { segmentFromShell, segmentOfHash, toShellPath } from "./sections";

/**
 * Keeps the app's MemoryRouter and the shell URL the same (contract §3: the
 * shell owns the address bar). In-app navigation → ctx.navigate("path#section")
 * (sections.ts — the hash is what the shell's sidebar sub-menu highlights);
 * back/forward in the shell → ctx.onRouteChange → the memory router follows
 * (replace). The sidebar sub-menu (manifest nav.items) changes only the hash
 * (hashchange) or, from another sub-screen, moves to "/apps/punisher#/map"
 * (onRouteChange("/") with that hash) — both open that tab like the old in-app
 * tabs did: another tab → its last address and scroll, the current tab → its
 * first screen. Equal paths are skipped both ways, so neither side echoes the
 * other. Shell-originated moves carry SHELL_NAV_STATE so the app treats them
 * like back/forward (no scroll-to-top, in-app history steps instead of growing)
 * and only fix the shell URL's hash in place (replace).
 */
export function RouteSync({ ctx }: { ctx: AppContext }) {
  const location = useLocation();
  const navigate = useNavigate();
  const type = useNavigationType();
  const first = useRef(true);
  const current = useRef("");
  current.current = `${location.pathname}${location.search}`;

  useEffect(() => {
    const target = toShellPath(`${location.pathname}${location.search}`);
    const shellNow = `${ctx.path}${ctx.search}${window.location.hash}`;
    if (first.current) {
      first.current = false;
      // 옛 주소(/apps/punisher/map)·하위 메뉴로 들어온 주소에 섹션 해시를 맞춘다(기록은 늘리지 않음).
      if (target !== shellNow) { ctx.navigate(target, { replace: true }); }
      return;
    }
    if (target === shellNow) { return; }
    // 첫 화면(POP — StrictMode 의 두 번째 실행 포함)·셸이 연 이동은 해시만 바로잡는다(replace).
    ctx.navigate(target, { replace: isShellNavigation(location.state) || type !== "PUSH" });
  }, [ctx, location.pathname, location.search, location.state, type]);

  useEffect(() => {
    const openTab = (segment: LearnSegment) => {
      const here = segmentOf(current.current.split("?")[0] ?? "");
      if (here) { rememberTab(here, current.current, window.scrollY); }
      const to = segment === here ? HREFS[segment] : tabHref(segment);
      const restoreScroll = segment === here ? 0 : tabScroll(segment) ?? 0;
      if (to !== current.current) { navigate(to, { replace: true, state: { ...SHELL_NAV_STATE, restoreScroll } }); }
    };
    const offRoute = ctx.onRouteChange((path, search) => {
      const segment = segmentFromShell(path, window.location.hash);
      if (segment) {
        openTab(segment);
        return;
      }
      // 앱이 ctx.navigate 로 바꾼 주소도 여기로 돌아온다 — 이미 그 자리면 다시 이동하지 않는다
      // (불필요한 REPLACE 가 쿼리만 바뀐 이동을 맨 위로 올리지 않게).
      if (`${path}${search}` === current.current) { return; }
      navigate(`${path}${search}`, { replace: true, state: SHELL_NAV_STATE });
    });
    const onHash = () => {
      const segment = segmentOfHash(window.location.hash);
      if (segment) { openTab(segment); }
    };
    window.addEventListener("hashchange", onHash);
    return () => {
      offRoute();
      window.removeEventListener("hashchange", onHash);
    };
  }, [ctx, navigate]);

  return null;
}

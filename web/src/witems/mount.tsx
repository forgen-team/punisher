import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import css from "@/src/styles/globals.css?inline";
import { AppProviders, AppRoutes } from "@/src/app/App";
import { setWitemsRuntime, witemsRuntime, type WitemsRuntime } from "@/src/shared/config/runtime";
import { ScrollToTopOnNavigate } from "@/src/shared/lib/router";
import type { AppContext, Unmount } from "./contract";
import { RouteSync } from "./RouteSync";
import { ROOT_SELECTOR } from "./scope";
import { WitemsShell } from "./WitemsShell";
import { witemsRoutes } from "./routes";
import { legacyHashPath } from "./legacyHash";

/**
 * MemoryRouter 첫 주소(basePath 뒤) = ctx.path + ctx.search. "/" 는 "" 로.
 * 옛 해시 주소(/apps/punisher#/map — D8 이전)로 들어오면 그 경로로 셸 주소를 바로잡는다(replace, 한 릴리스용).
 */
function initialEntry(ctx: AppContext): string {
  const legacy = legacyHashPath(ctx.path, ctx.search, window.location.hash);
  if (legacy) { ctx.navigate(legacy, { replace: true }); }
  const path = legacy ?? `${ctx.path}${ctx.search}`;
  return path.startsWith("/?") || path === "/" ? path.slice(1) : path;
}

const ROOT_CLASS = ROOT_SELECTOR.slice(1);

/**
 * Witems contract §3: mount(el, ctx) → unmount. Styles go into the Shadow DOM
 * (§4); el and the portal carry the root class, data-theme and
 * data-seed-color-mode so the shell's tokens and dark mode apply to both.
 */
export function mount(el: HTMLElement, ctx: AppContext): Unmount {
  const runtime: WitemsRuntime = {
    apiBase: ctx.apiBase,
    portal: ctx.portal,
    setTitle: (title) => ctx.setTitle(title),
    // 구버전 셸에는 setActions 가 없다 — 그때는 undefined 로 둔다(셸이 페이지 동작을 그리지 않는다).
    setActions: typeof ctx.setActions === "function" ? (actions) => ctx.setActions?.(actions) : undefined,
    setCommands: typeof ctx.setCommands === "function" ? (commands) => ctx.setCommands?.(commands) : undefined,
    // 섹션 메뉴(manifest nav.items) 숫자 — 키는 섹션 경로("/"). 셸 안에서는 섹션 메뉴를 셸이 그린다(LearnNav 없음).
    setBadges: typeof ctx.setBadges === "function" ? (badges) => ctx.setBadges?.(badges) : undefined,
  };
  setWitemsRuntime(runtime);

  const style = document.createElement("style");
  style.textContent = css;
  const shadow = el.getRootNode();
  if (shadow instanceof ShadowRoot) {
    shadow.prepend(style);
  } else {
    el.prepend(style);
  }

  const surfaces = [el, ctx.portal];
  const applyTheme = (theme: string) => {
    for (const node of surfaces) {
      node.classList.add(ROOT_CLASS);
      node.setAttribute("data-theme", theme);
      // SEED 부품의 연결하지 않은 값(비활성 색 등)도 같은 모드로(@forgen-team/ui README)
      node.setAttribute("data-seed-color-mode", theme === "dark" ? "dark-only" : "light-only");
    }
  };
  applyTheme(ctx.theme);
  const offTheme = ctx.onThemeChange(applyTheme);

  const root = createRoot(el);
  root.render(
    <StrictMode>
      <AppProviders inShell portal={ctx.portal}>
        <MemoryRouter basename={ctx.basePath} initialEntries={[`${ctx.basePath}${initialEntry(ctx)}`]}>
          <RouteSync ctx={ctx} />
          <ScrollToTopOnNavigate />
          <AppRoutes shell={<WitemsShell />} extra={witemsRoutes()} />
        </MemoryRouter>
      </AppProviders>
    </StrictMode>,
  );

  return () => {
    runtime.setActions?.(null);
    runtime.setCommands?.(null);
    offTheme();
    root.unmount();
    style.remove();
    for (const node of surfaces) {
      node.classList.remove(ROOT_CLASS);
      node.removeAttribute("data-theme");
      node.removeAttribute("data-seed-color-mode");
    }
    // 다시 시도 등으로 새 mount 가 먼저 runtime 을 바꿨으면 그대로 둔다
    if (witemsRuntime() === runtime) { setWitemsRuntime(null); }
  };
}

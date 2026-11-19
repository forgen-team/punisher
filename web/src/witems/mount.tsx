import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router-dom";
import css from "@/src/styles/globals.css?inline";
import { AppProviders, AppRoutes } from "@/src/app/App";
import { setWitemsRuntime } from "@/src/shared/config/runtime";
import { ScrollToTopOnNavigate } from "@/src/shared/lib/router";
import type { AppContext, Unmount } from "./contract";
import { RouteSync } from "./RouteSync";
import { ROOT_SELECTOR } from "./scope";
import { WitemsShell } from "./WitemsShell";
import { witemsRoutes } from "./routes";

const ROOT_CLASS = ROOT_SELECTOR.slice(1);

/**
 * Witems contract §3: mount(el, ctx) → unmount. Styles go into the Shadow DOM
 * (§4); el and the portal carry the root class and data-theme so the design
 * tokens and dark mode apply to both.
 */
export function mount(el: HTMLElement, ctx: AppContext): Unmount {
  setWitemsRuntime({ apiBase: ctx.apiBase, portal: ctx.portal, setTitle: (title) => ctx.setTitle(title) });

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
    }
  };
  applyTheme(ctx.theme);
  const offTheme = ctx.onThemeChange(applyTheme);

  const root = createRoot(el);
  root.render(
    <StrictMode>
      <AppProviders>
        <MemoryRouter basename={ctx.basePath} initialEntries={[`${ctx.basePath}${ctx.path === "/" ? "" : ctx.path}${ctx.search}`]}>
          <RouteSync ctx={ctx} />
          <ScrollToTopOnNavigate />
          <AppRoutes shell={<WitemsShell />} extra={witemsRoutes()} />
        </MemoryRouter>
      </AppProviders>
    </StrictMode>,
  );

  return () => {
    offTheme();
    root.unmount();
    style.remove();
    for (const node of surfaces) {
      node.classList.remove(ROOT_CLASS);
      node.removeAttribute("data-theme");
    }
    setWitemsRuntime(null);
  };
}

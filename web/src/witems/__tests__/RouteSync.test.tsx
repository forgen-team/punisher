import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation } from "react-router-dom";
import type { AppContext } from "../contract";
import { RouteSync } from "../RouteSync";

const BASE = "/apps/punisher";

/** 셸 흉내: navigate 는 주소(경로+해시)를 바꾸고 경로가 바뀌면 onRouteChange 를 알린다(AppHost 처럼). */
function fakeShell(start: string) {
  const listeners = new Set<(path: string, search: string) => void>();
  const split = (url: string) => {
    const parsed = new URL(url, "http://x");
    const path = parsed.pathname.slice(BASE.length) || "/";
    return { path, search: parsed.search, hash: parsed.hash };
  };
  let now = split(`${BASE}${start}`);
  window.history.replaceState(null, "", `${BASE}${start}`);
  const go = (url: string) => {
    const next = split(url);
    window.history.replaceState(null, "", url);
    const moved = next.path !== now.path || next.search !== now.search;
    now = next;
    if (moved) { listeners.forEach((cb) => cb(next.path, next.search)); }
  };
  const navigate = vi.fn((to: string) => go(`${BASE}${to.startsWith("/#") || to.startsWith("/?") ? to.slice(1) : to === "/" ? "" : to}`));
  const ctx = {
    get path() { return now.path; },
    get search() { return now.search; },
    onRouteChange: (cb: (path: string, search: string) => void) => {
      listeners.add(cb);
      return () => listeners.delete(cb);
    },
    navigate,
  } as unknown as AppContext;
  return { ctx, navigate, go };
}

let seen = "";
function Where() {
  const { pathname, search } = useLocation();
  seen = `${pathname}${search}`;
  return null;
}

function mountAt(shell: ReturnType<typeof fakeShell>, initial: string) {
  return render(
    <MemoryRouter basename={BASE} initialEntries={[`${BASE}${initial === "/" ? "" : initial}`]}>
      <RouteSync ctx={shell.ctx} />
      <Where />
    </MemoryRouter>,
  );
}

describe("RouteSync — shell sidebar sub-menu", () => {
  beforeEach(() => { seen = ""; });
  afterEach(() => window.history.replaceState(null, "", "/"));

  it("adds the section hash to the shell URL on mount without a new entry", () => {
    const shell = fakeShell("/map");
    mountAt(shell, "/map");
    expect(shell.navigate).toHaveBeenCalledWith("/map#/map", { replace: true });
    expect(window.location.pathname + window.location.hash).toBe(`${BASE}/map#/map`);
  });

  it("opens the tab when the shell changes only the hash (same app screen)", () => {
    const shell = fakeShell("/#/");
    mountAt(shell, "/");
    act(() => {
      window.location.hash = "#/history";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(seen).toBe("/history");
    expect(window.location.pathname + window.location.hash).toBe(`${BASE}/history#/history`);
  });

  it("opens the tab when the shell moves to base#hash from another sub-screen", () => {
    const shell = fakeShell("/map#/map");
    mountAt(shell, "/map");
    act(() => shell.go(`${BASE}#/history`));
    expect(seen).toBe("/history");
    expect(window.location.pathname + window.location.hash).toBe(`${BASE}/history#/history`);
  });

  it("follows back/forward to a path whose hash agrees", () => {
    const shell = fakeShell("/history#/history");
    mountAt(shell, "/history");
    act(() => shell.go(`${BASE}/lectures/l1#/lectures`));
    expect(seen).toBe("/lectures/l1");
    act(() => shell.go(`${BASE}/settings`));
    expect(seen).toBe("/settings");
  });
});

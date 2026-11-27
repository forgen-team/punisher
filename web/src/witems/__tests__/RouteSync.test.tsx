import { act, render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation, useNavigate } from "react-router-dom";
import type { AppContext } from "../contract";
import { RouteSync } from "../RouteSync";

const BASE = "/apps/punisher";

/** 셸 흉내: navigate·섹션 메뉴(go)는 경로를 바꾸고, 바뀌면 onRouteChange 로 알린다(AppHost 처럼, 다시 mount 없음). */
function fakeShell(start: string) {
  const listeners = new Set<(path: string, search: string) => void>();
  const split = (url: string) => {
    const parsed = new URL(url, "http://x");
    return { path: parsed.pathname.slice(BASE.length) || "/", search: parsed.search };
  };
  let now = split(`${BASE}${start}`);
  const go = (url: string) => {
    const next = split(url);
    const moved = next.path !== now.path || next.search !== now.search;
    now = next;
    if (moved) { listeners.forEach((cb) => cb(next.path, next.search)); }
  };
  const navigate = vi.fn((to: string) => go(`${BASE}${to === "/" ? "" : to}`));
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
let appNavigate: ReturnType<typeof useNavigate> | null = null;
function Where() {
  const { pathname, search } = useLocation();
  appNavigate = useNavigate();
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

describe("RouteSync — path sections (D8)", () => {
  beforeEach(() => { seen = ""; appNavigate = null; });

  it("does not touch the shell URL on mount", () => {
    const shell = fakeShell("/map");
    mountAt(shell, "/map");
    expect(shell.navigate).not.toHaveBeenCalled();
  });

  it("follows the shell section menu without a hash", () => {
    const shell = fakeShell("/");
    mountAt(shell, "/");
    act(() => shell.go(`${BASE}/history?area=net`));
    expect(seen).toBe("/history?area=net");
    act(() => shell.go(`${BASE}`));
    expect(seen).toBe("/");
    expect(shell.navigate).not.toHaveBeenCalled();
  });

  it("sends in-app moves to the shell as plain paths", () => {
    const shell = fakeShell("/");
    mountAt(shell, "/");
    act(() => appNavigate?.("/lectures/l1?at=p3"));
    expect(shell.navigate).toHaveBeenLastCalledWith("/lectures/l1?at=p3", { replace: false });
    expect(seen).toBe("/lectures/l1?at=p3");
  });
});

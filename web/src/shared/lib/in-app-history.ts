import { useEffect, useRef } from "react";
import { useLocation, useNavigationType } from "react-router-dom";

/**
 * 앱 안 이동 기록 — 하위 화면의 ← 가 "들어온 곳으로(브라우저 뒤로)" 갈지 "기본 상위 화면으로" 갈지 정한다.
 * 단순히 이동 횟수를 세면 뒤로 간 것도 하나로 세어, 직접 들어온 첫 화면으로 돌아와서도 ← 가 앱 밖으로 나갔다.
 * 그래서 방문 줄(entries)과 지금 자리(index)를 둔다: 새 이동은 뒤를 자르고 붙이고, 교체는 지금 자리를 바꾸고,
 * 뒤로/앞으로(단독판 POP, Witems 는 셸이 알린 이동)는 바로 앞·뒤 칸과 같으면 그쪽으로 옮긴다(모르면 새로 시작).
 */
export interface NavStack {
  entries: string[];
  index: number;
}

export type NavKind = "push" | "replace" | "pop";

export function nextNavStack(stack: NavStack, kind: NavKind, entry: string): NavStack {
  const { entries, index } = stack;
  if (kind === "push") {
    return { entries: [...entries.slice(0, index + 1), entry], index: index + 1 };
  }
  if (kind === "replace") {
    const next = entries.slice();
    next[index] = entry;
    return { entries: next, index };
  }
  if (entries[index] === entry) { return stack; }
  if (index > 0 && entries[index - 1] === entry) { return { entries, index: index - 1 }; }
  if (entries[index + 1] === entry) { return { entries, index: index + 1 }; }
  return { entries: [entry], index: 0 };
}

/** 새로고침 뒤에도 같은 판단을 하도록 sessionStorage 에도 남긴다. */
const IN_APP_NAV_KEY = "punisher:in-app-nav";
const MAX_ENTRIES = 50;

let stack: NavStack = { entries: [], index: 0 };

/** 앱 안에서 들어왔는가(브라우저 뒤로가기로 들어온 곳에 돌아갈 수 있는가). */
export function canGoBack(): boolean {
  return stack.index > 0;
}

function restore(entry: string): NavStack {
  try {
    const raw = window.sessionStorage.getItem(IN_APP_NAV_KEY);
    const saved = raw ? (JSON.parse(raw) as Partial<NavStack>) : null;
    if (saved && Array.isArray(saved.entries) && typeof saved.index === "number" && saved.entries[saved.index] === entry) {
      return { entries: saved.entries.filter((e): e is string => typeof e === "string"), index: saved.index };
    }
  } catch {
    // 저장소를 못 쓰거나 값이 깨졌으면 새로 시작한다
  }
  return { entries: [entry], index: 0 };
}

function persist(next: NavStack) {
  const drop = Math.max(0, next.entries.length - MAX_ENTRIES);
  stack = drop > 0 ? { entries: next.entries.slice(drop), index: Math.max(0, next.index - drop) } : next;
  try {
    window.sessionStorage.setItem(IN_APP_NAV_KEY, JSON.stringify(stack));
  } catch {
    // 저장소를 못 쓰면 이번 탭의 메모리 기록만 쓴다
  }
}

/** Witems 셸이 알린 이동(뒤로/앞으로 등)에 RouteSync 가 다는 표지 — 앱이 만든 이동과 구분한다. */
export const SHELL_NAV_STATE = { fromShell: true } as const;

export function isShellNavigation(state: unknown): boolean {
  return typeof state === "object" && state !== null && (state as { fromShell?: unknown }).fromShell === true;
}

/** 셸(AppShell·WitemsShell)이 한 번 호출한다. */
export function useTrackInAppNavigation() {
  const location = useLocation();
  const type = useNavigationType();
  const first = useRef(true);
  // 효과는 location.key 가 바뀔 때만(같은 주소로 다시 이동해도 한 번씩) — 나머지 값은 ref 로 읽는다.
  const latest = useRef({ entry: "", kind: "replace" as NavKind });
  latest.current.entry = `${location.pathname}${location.search}`;
  if (type === "PUSH") { latest.current.kind = "push"; }
  else if (type === "POP" || isShellNavigation(location.state)) { latest.current.kind = "pop"; }
  else { latest.current.kind = "replace"; }
  const { key } = location;
  useEffect(() => {
    const { entry, kind } = latest.current;
    if (first.current) {
      first.current = false;
      persist(restore(entry));
      return;
    }
    persist(nextNavStack(stack, kind, entry));
  }, [key]);
}

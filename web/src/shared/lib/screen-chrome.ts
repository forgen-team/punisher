import { useEffect, useLayoutEffect, useRef, useSyncExternalStore } from "react";
import type { ShellAction } from "@/src/shared/config/runtime";

/**
 * 화면이 셸에 넘기는 머리 정보 — 구체적인 제목(강의 제목 등)·부제(오늘 날짜 등)·페이지 동작(이동·설정). 셸(WitemsShell·AppShell)이
 * 읽어 Witems 는 ctx.setTitle·ctx.setActions 로, 단독판은 document.title 로 보낸다.
 * 부제는 단독판 Page subtitle 로만 그린다(셸에 부제를 넘기는 계약이 아직 없어 셸 안에서는 보이지 않는다).
 * 자식 화면의 effect 가 부모 셸보다 먼저 돌아 셸이 덮어쓰지 않게 값을 여기 모은다.
 */
export interface PageAction extends Omit<ShellAction, "kind"> {
  kind?: ShellAction["kind"];
}

interface Chrome {
  title: string | null;
  subtitle: string | null;
  actions: PageAction[] | null;
}

let chrome: Chrome = { title: null, subtitle: null, actions: null };
const listeners = new Set<() => void>();

function set(next: Partial<Chrome>) {
  chrome = { ...chrome, ...next };
  for (const listener of listeners) { listener(); }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}

export function useScreenChrome(): Chrome {
  return useSyncExternalStore(subscribe, () => chrome, () => chrome);
}

/** 이 화면의 구체적인 제목(null·빈 값 = 셸의 경로별 기본 제목). 화면이 내려가면 지운다. */
export function useScreenTitle(title: string | null | undefined) {
  const value = title?.trim() ? title.trim() : null;
  useLayoutEffect(() => {
    set({ title: value });
  }, [value]);
  useLayoutEffect(() => () => set({ title: null }), []);
}

/** 이 화면의 부제(제목 아래 한 줄 — 오늘 날짜 등). 화면이 내려가면 지운다. */
export function useScreenSubtitle(subtitle: string | null | undefined) {
  const value = subtitle?.trim() ? subtitle.trim() : null;
  useLayoutEffect(() => {
    set({ subtitle: value });
  }, [value]);
  useLayoutEffect(() => () => set({ subtitle: null }), []);
}

/** 동작 목록을 바꿀 필요가 있는지 비교하는 열쇠(onSelect 는 매 렌더 새 함수라 뺀다). */
function actionsKey(actions: PageAction[] | null): string {
  return actions ? JSON.stringify(actions.map(({ onSelect, ...rest }) => ({ ...rest, run: Boolean(onSelect) }))) : "";
}

/**
 * 이 화면의 페이지 동작(화면 전체에 대한 이동·설정). 셸이 그린다 — Witems 상단 바 ⋯·제목 줄 버튼.
 * onSelect 는 마지막 렌더의 것을 부른다(목록이 같으면 셸에 다시 보내지 않는다).
 */
export function usePageActions(actions: PageAction[] | null) {
  const latest = useRef(actions);
  latest.current = actions;
  const key = actionsKey(actions);
  useLayoutEffect(() => {
    const current = latest.current;
    set({
      actions: current
        ? current.map((a) => (a.onSelect ? { ...a, onSelect: () => latest.current?.find((x) => x.id === a.id)?.onSelect?.() } : a))
        : null,
    });
  }, [key]);
  useEffect(() => () => set({ actions: null }), []);
}

export function actionsSignature(actions: PageAction[] | null): string {
  return actionsKey(actions);
}

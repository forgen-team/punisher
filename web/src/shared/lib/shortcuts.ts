import { useEffect, useRef } from "react";
import { eventOrigin } from "./dom-root";
import { useUiPrefs } from "./ui-prefs";

/**
 * 복습 단축키(설정에서 끌 수 있음): 키 → 동작. 입력란·버튼 위에서 친 키, 조합키(⌘·Ctrl·Alt), 한글 조합 중인 키는 무시한다.
 * Shadow DOM 안이면 window 에서 본 target 이 호스트이므로 composedPath 로 실제 요소를 본다.
 */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) { return false; }
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable;
}

export function useStudyShortcuts(map: Record<string, (() => void) | undefined>, active = true) {
  const { shortcuts } = useUiPrefs();
  const latest = useRef(map);
  latest.current = map;
  const on = active && shortcuts;
  useEffect(() => {
    if (!on) { return undefined; }
    const onKey = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey || event.isComposing || event.repeat) { return; }
      const origin = eventOrigin(event);
      if (isTypingTarget(origin)) { return; }
      // Space·Enter 가 포커스된 버튼·링크를 누르는 기본 동작과 겹치지 않게, 그 위에서는 단축키로 쓰지 않는다.
      if ((event.key === " " || event.key === "Enter") && origin instanceof HTMLElement && origin.closest("button, a, [role='button']")) { return; }
      const run = latest.current[event.key];
      if (!run) { return; }
      event.preventDefault();
      run();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [on]);
  return on;
}

/** ⌘K / Ctrl+K → 학습 자료 검색(셸 둘 다). 단축키 설정과 무관(조합키라 입력과 겹치지 않는다). */
export function useSearchShortcut(open: () => void) {
  const latest = useRef(open);
  latest.current = open;
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && !event.altKey && !event.shiftKey && event.key.toLowerCase() === "k") {
        event.preventDefault();
        latest.current();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
}

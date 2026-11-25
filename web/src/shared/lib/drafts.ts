import { useCallback, useEffect, useRef, useState } from "react";

/**
 * 미저장 편집의 초안 자동 보관(localStorage) — 다른 화면에 갔다 오거나 새로고침해도 이어서 쓴다.
 * 저장된 초안은 그 초안을 시작한 원본(base)과 함께 둔다: 원본이 바뀌었으면(다른 곳에서 저장) 옛 초안은 버린다.
 */
const PREFIX = "punisher:draft:";
const WRITE_DELAY_MS = 400;
/** 오래된 초안은 쓰지 않는다(30일). */
const MAX_AGE_MS = 30 * 24 * 60 * 60 * 1000;

interface Stored<T> {
  base: string;
  value: T;
  savedAt: number;
}

function storage(): Storage | null {
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

export function readDraft<T>(key: string, base = ""): T | null {
  const raw = storage()?.getItem(PREFIX + key);
  if (!raw) { return null; }
  try {
    const stored = JSON.parse(raw) as Stored<T>;
    if (stored.base !== base || Date.now() - stored.savedAt > MAX_AGE_MS) {
      clearDraft(key);
      return null;
    }
    return stored.value;
  } catch {
    clearDraft(key);
    return null;
  }
}

export function writeDraft<T>(key: string, value: T, base = ""): void {
  try {
    storage()?.setItem(PREFIX + key, JSON.stringify({ base, value, savedAt: Date.now() } satisfies Stored<T>));
  } catch (err) {
    // 저장 공간이 가득 찼거나 막혔으면 이번 화면의 상태만 쓴다
    console.warn("punisher: draft not saved", err);
  }
}

export function clearDraft(key: string): void {
  try {
    storage()?.removeItem(PREFIX + key);
  } catch (err) {
    console.warn("punisher: draft not cleared", err);
  }
}

export interface DraftState<T> {
  value: T;
  setValue: (next: T | ((prev: T) => T)) => void;
  /** 저장된 초안을 불러와 시작했는가(안내 한 줄용) */
  restored: boolean;
  /** 초안을 버리고 처음 값으로 */
  discard: () => void;
  /** 저장(보내기)이 끝났을 때 — 보관만 지우고 값은 그대로 */
  clear: () => void;
}

/**
 * useState 처럼 쓰되 값이 initial 과 다르면 key 로 보관한다. base 는 초안의 원본(예: 저장된 프롬프트 본문).
 * isEmpty 가 참인 값(기본: initial 과 같음)은 보관하지 않는다.
 */
export function useDraftState<T>(key: string, initial: T, base = "", isEmpty?: (value: T) => boolean): DraftState<T> {
  const emptyRef = useRef(isEmpty);
  emptyRef.current = isEmpty;
  const initialRef = useRef(initial);
  initialRef.current = initial;
  const [state, setState] = useState(() => {
    const saved = readDraft<T>(key, base);
    return { key, base, value: saved ?? initial, restored: saved !== null };
  });
  // key·base 가 바뀌면(다른 블록) 그 초안으로 다시 시작한다.
  let current = state;
  if (state.key !== key || state.base !== base) {
    const saved = readDraft<T>(key, base);
    current = { key, base, value: saved ?? initial, restored: saved !== null };
    setState(current);
  }

  const value = current.value;
  useEffect(() => {
    const empty = emptyRef.current ? emptyRef.current(value) : JSON.stringify(value) === JSON.stringify(initialRef.current);
    const timer = setTimeout(() => (empty ? clearDraft(key) : writeDraft(key, value, base)), WRITE_DELAY_MS);
    return () => clearTimeout(timer);
  }, [key, base, value]);

  const setValue = useCallback((next: T | ((prev: T) => T)) => {
    setState((prev) => ({ ...prev, value: typeof next === "function" ? (next as (p: T) => T)(prev.value) : next }));
  }, []);
  const discard = useCallback(() => {
    clearDraft(key);
    setState((prev) => ({ ...prev, value: initialRef.current, restored: false }));
  }, [key]);
  const clear = useCallback(() => {
    clearDraft(key);
    setState((prev) => ({ ...prev, restored: false }));
  }, [key]);

  return { value, setValue, restored: current.restored, discard, clear };
}

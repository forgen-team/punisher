import { useSyncExternalStore } from "react";

/**
 * 이 기기의 화면 선호 — 복습 단축키, 강의 글자 크기·줄 폭(localStorage). 서버 설정이 아니라 기기마다 다르게 둘 수 있다
 * (폰은 크게, 넓은 화면은 좁은 줄 등).
 */
export type FontSize = "sm" | "md" | "lg";
export type LineWidth = "narrow" | "normal" | "wide";

export interface UiPrefs {
  shortcuts: boolean;
  fontSize: FontSize;
  lineWidth: LineWidth;
}

export const DEFAULT_UI_PREFS: UiPrefs = { shortcuts: true, fontSize: "md", lineWidth: "normal" };

const KEY = "punisher.uiPrefs";
const listeners = new Set<() => void>();
let cached: UiPrefs | null = null;

function read(): UiPrefs {
  if (cached) { return cached; }
  let next = DEFAULT_UI_PREFS;
  try {
    const raw = window.localStorage.getItem(KEY);
    if (raw) { next = sanitize(JSON.parse(raw) as Partial<UiPrefs>); }
  } catch {
    // 저장소를 못 쓰거나 값이 깨졌으면 기본값
  }
  cached = next;
  return next;
}

export function sanitize(v: Partial<UiPrefs>): UiPrefs {
  return {
    shortcuts: typeof v.shortcuts === "boolean" ? v.shortcuts : DEFAULT_UI_PREFS.shortcuts,
    fontSize: v.fontSize === "sm" || v.fontSize === "lg" || v.fontSize === "md" ? v.fontSize : DEFAULT_UI_PREFS.fontSize,
    lineWidth: v.lineWidth === "narrow" || v.lineWidth === "wide" || v.lineWidth === "normal" ? v.lineWidth : DEFAULT_UI_PREFS.lineWidth,
  };
}

export function setUiPrefs(patch: Partial<UiPrefs>): void {
  cached = sanitize({ ...read(), ...patch });
  try {
    window.localStorage.setItem(KEY, JSON.stringify(cached));
  } catch (err) {
    console.warn("punisher: ui prefs not saved", err);
  }
  for (const l of listeners) { l(); }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}

export function useUiPrefs(): UiPrefs {
  return useSyncExternalStore(subscribe, read, () => DEFAULT_UI_PREFS);
}

/** 강의 본문에 거는 CSS 변수(LectureLayout): 글자 크기 배율·본문 최대 폭. */
export const READING_VARS: Record<FontSize | LineWidth, Record<string, string>> = {
  sm: { "--reading-scale": "0.9375" },
  md: { "--reading-scale": "1" },
  lg: { "--reading-scale": "1.125" },
  narrow: { "--reading-measure": "34rem" },
  normal: { "--reading-measure": "42rem" },
  wide: { "--reading-measure": "52rem" },
};

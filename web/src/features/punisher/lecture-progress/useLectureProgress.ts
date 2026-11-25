
import { useCallback, useEffect, useRef } from "react";
import { useLocation } from "react-router-dom";
import { punisherApi, type LectureProgress } from "@/src/shared/api/punisher";
import { anchorFromLocation } from "@/src/entities/punisher";
import { getAppElementById, queryAppAll } from "@/src/shared/lib/dom-root";

const SAVE_DEBOUNCE_MS = 2000;
const TOP_OFFSET_PX = 120;

export function anchorElementId(anchor: string): string {
  return `pa-${anchor}`;
}

export function scrollToAnchor(anchor: string, behavior: ScrollBehavior = "smooth"): boolean {
  // Witems 는 본문을 Shadow DOM 에 그린다 — document 가 아니라 화면 루트에서 찾는다.
  const el = getAppElementById(anchorElementId(anchor));
  if (!el) { return false; }
  el.scrollIntoView({ behavior, block: "start" });
  return true;
}

/** 바텀시트의 스크롤 잠금(body position:fixed + top:-Ypx) 중에도 실제 읽던 위치를 돌려준다. */
function isScrollLocked(): boolean {
  return document.body.style.position === "fixed";
}

function currentScrollY(): number {
  if (isScrollLocked()) { return Math.abs(Number.parseFloat(document.body.style.top) || 0); }
  return window.scrollY;
}

function readPosition(): { anchor: string; percent: number } {
  const nodes = queryAppAll<HTMLElement>("[data-lecture-anchor]");
  let anchor = "";
  for (const node of nodes) {
    if (node.getBoundingClientRect().top <= TOP_OFFSET_PX) { anchor = node.dataset.lectureAnchor ?? anchor; }
    else { break; }
  }
  if (!anchor && nodes[0]) { anchor = nodes[0].dataset.lectureAnchor ?? ""; }
  const scrollable = document.documentElement.scrollHeight - window.innerHeight;
  const percent = scrollable > 0 ? Math.round((currentScrollY() / scrollable) * 100) : 100;
  return { anchor, percent: Math.min(100, Math.max(0, percent)) };
}

/**
 * Tracks the reading anchor on scroll, saves it with a debounced PUT, flushes on
 * page hide/unmount, and resumes to the saved anchor once the content is ready.
 */
export function useLectureProgress(lectureId: string, initial: LectureProgress | undefined, ready: boolean) {
  const lastSaved = useRef({ anchor: initial?.anchor ?? "", percent: initial?.percent ?? 0 });
  const current = useRef(lastSaved.current);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const resumed = useRef(false);
  const initialAnchor = initial?.anchor ?? "";
  const { search, hash } = useLocation();
  const linkAnchor = anchorFromLocation(search, hash);

  const flush = useCallback(() => {
    if (timer.current) { clearTimeout(timer.current); }
    timer.current = null;
    const pos = current.current;
    if (!pos.anchor) { return; }
    if (pos.anchor === lastSaved.current.anchor && pos.percent === lastSaved.current.percent) { return; }
    lastSaved.current = pos;
    punisherApi.saveProgress(lectureId, pos.anchor, pos.percent).catch((err: unknown) => {
      console.warn("punisher: progress save failed", err);
    });
  }, [lectureId]);

  useEffect(() => {
    if (!ready) { return; }
    let frame = 0;
    function onScroll() {
      // 시트가 열려 body가 고정되는 순간의 scroll 이벤트(위치 0)는 읽기 진행이 아니다.
      if (isScrollLocked()) { return; }
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        current.current = readPosition();
        if (timer.current) { clearTimeout(timer.current); }
        timer.current = setTimeout(flush, SAVE_DEBOUNCE_MS);
      });
    }
    function onHide() {
      if (document.visibilityState === "hidden") { flush(); }
    }
    window.addEventListener("scroll", onScroll, { passive: true });
    document.addEventListener("visibilitychange", onHide);
    window.addEventListener("pagehide", flush);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("scroll", onScroll);
      document.removeEventListener("visibilitychange", onHide);
      window.removeEventListener("pagehide", flush);
      flush();
    };
  }, [ready, flush]);

  useEffect(() => {
    if (!ready || resumed.current) { return; }
    resumed.current = true;
    // 링크로 받은 자리(?at=, 옛 #pa-)가 저장된 진도보다 앞선다.
    const target = linkAnchor || initialAnchor;
    if (target) { scrollToAnchor(target, "auto"); }
  }, [ready, initialAnchor, linkAnchor]);

  /** Current anchor (for questions, notes, feedback, bookmarks). */
  const getAnchor = useCallback(() => current.current.anchor || readPosition().anchor, []);

  /** Saves immediately (e.g. before "더 깊게" navigation). */
  const saveNow = useCallback(() => {
    current.current = readPosition();
    flush();
  }, [flush]);

  return { getAnchor, saveNow };
}

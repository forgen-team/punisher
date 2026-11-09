
import { useCallback, useEffect, useRef } from "react";
import { punisherApi, type LectureProgress } from "@/src/shared/api/punisher";

const SAVE_DEBOUNCE_MS = 2000;
const TOP_OFFSET_PX = 120;

export function anchorElementId(anchor: string): string {
  return `pa-${anchor}`;
}

export function scrollToAnchor(anchor: string, behavior: ScrollBehavior = "smooth"): boolean {
  const el = document.getElementById(anchorElementId(anchor));
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
  const nodes = document.querySelectorAll<HTMLElement>("[data-lecture-anchor]");
  let anchor = "";
  for (const node of Array.from(nodes)) {
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
    const hash = window.location.hash;
    const hashAnchor = hash.startsWith("#pa-") ? decodeURIComponent(hash.slice(4)) : "";
    const target = hashAnchor || initialAnchor;
    if (target) { scrollToAnchor(target, "auto"); }
  }, [ready, initialAnchor]);

  /** Current anchor (for questions, notes, feedback, bookmarks). */
  const getAnchor = useCallback(() => current.current.anchor || readPosition().anchor, []);

  /** Saves immediately (e.g. before "더 깊게" navigation). */
  const saveNow = useCallback(() => {
    current.current = readPosition();
    flush();
  }, [flush]);

  return { getAnchor, saveNow };
}

import { useEffect, useMemo, useRef, type ReactNode, type RefObject } from "react";
import { punisherApi } from "@/src/shared/api/punisher";
import { queryAppAll } from "@/src/shared/lib/dom-root";
import { ReadingSignalsContext, type ReadingSignalsRecorder } from "@/src/shared/lib/readingSignals";
import { BLOCK_ANCHOR, SignalBuffer, focusedAnchor } from "./lib/buffer";

const TICK_MS = 1000;
const RESCAN_MS = 5000;
const FLUSH_MS = 30_000;
/** 이만큼 입력(스크롤·키·포인터)이 없으면 자리를 비운 것으로 보고 체류 시간을 세지 않는다. */
const IDLE_MS = 120_000;

interface ReadingSignalsProviderProps {
  lectureId: string;
  /** 강의 본문을 감싼 요소 — 그 안의 블록 앵커만 본다(Witems Shadow DOM 도 같은 방법). */
  containerRef: RefObject<HTMLElement | null>;
  children: ReactNode;
}

/**
 * 블록 체류 시간과 용어 팝오버 열기를 모아 30초마다(화면을 떠날 때도) 보낸다(조사 #15).
 * 화면 중심에 가장 많이 보이는 블록 하나에 1초씩 쌓고, 탭이 가려졌거나 2분 동안 입력이 없으면 세지 않는다.
 */
export function ReadingSignalsProvider({ lectureId, containerRef, children }: ReadingSignalsProviderProps) {
  const buffer = useRef(new SignalBuffer());
  const recorder = useMemo<ReadingSignalsRecorder>(
    () => ({ termOpened: (conceptId, anchor) => { if (anchor && conceptId) { buffer.current.term(anchor, conceptId); } } }),
    [],
  );

  useEffect(() => {
    const signals = buffer.current;
    const flush = (keepalive: boolean) => {
      const batch = signals.drain();
      if (batch.length > 0) { void punisherApi.postReadingSignals(lectureId, batch, keepalive).catch(() => undefined); }
    };
    if (typeof IntersectionObserver === "undefined") { return () => flush(true); }
    const shares = new Map<string, number>();
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        const anchor = (entry.target as HTMLElement).dataset.lectureAnchor ?? "";
        if (!entry.isIntersecting) { shares.delete(anchor); continue; }
        const viewport = entry.rootBounds?.height || window.innerHeight || 1;
        shares.set(anchor, Math.max(entry.intersectionRatio, entry.intersectionRect.height / viewport));
      }
    }, { threshold: [0, 0.25, 0.5, 0.75, 1] });
    const observed = new WeakSet<Element>();
    const scan = () => {
      const root = containerRef.current;
      if (!root) { return; }
      for (const el of queryAppAll<HTMLElement>("[data-lecture-anchor]", root)) {
        if (!observed.has(el) && root.contains(el) && BLOCK_ANCHOR.test(el.dataset.lectureAnchor ?? "")) {
          observed.add(el);
          observer.observe(el);
        }
      }
    };
    let lastInput = Date.now();
    const onInput = () => { lastInput = Date.now(); };
    const onVisibility = () => { if (document.visibilityState === "hidden") { flush(true); } };
    let elapsed = 0;
    scan();
    const timer = window.setInterval(() => {
      elapsed += TICK_MS;
      if (elapsed % RESCAN_MS === 0) { scan(); }
      if (elapsed % FLUSH_MS === 0) { flush(false); }
      if (document.visibilityState !== "visible" || Date.now() - lastInput > IDLE_MS) { return; }
      const anchor = focusedAnchor(shares);
      if (anchor) { signals.dwell(anchor, TICK_MS); }
    }, TICK_MS);
    const opts = { capture: true, passive: true } as const;
    window.addEventListener("scroll", onInput, opts);
    window.addEventListener("keydown", onInput, opts);
    window.addEventListener("pointermove", onInput, opts);
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      window.clearInterval(timer);
      observer.disconnect();
      window.removeEventListener("scroll", onInput, opts);
      window.removeEventListener("keydown", onInput, opts);
      window.removeEventListener("pointermove", onInput, opts);
      document.removeEventListener("visibilitychange", onVisibility);
      flush(true);
    };
  }, [lectureId, containerRef]);

  return <ReadingSignalsContext.Provider value={recorder}>{children}</ReadingSignalsContext.Provider>;
}

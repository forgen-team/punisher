import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { createPortal } from "react-dom";
import { ChevronDown } from "lucide-react";
import { IconButton, cn, focusRing, useFocusTrap, useMediaQuery } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { placeOverlay, type OverlayPlacement } from "./overlayPlacement";
import { portalRoot } from "@/src/shared/config/runtime";
import { eventWithin } from "@/src/shared/lib/dom-root";

/** 읽기 위치를 기준으로 붙일 사각형(문서 좌표가 아니라 getBoundingClientRect 값). */
export interface OverlayAnchor {
  top: number;
  bottom: number;
  left: number;
  right: number;
}

export function toOverlayAnchor(rect: DOMRect): OverlayAnchor {
  return { top: rect.top, bottom: rect.bottom, left: rect.left, right: rect.right };
}

interface ReadingOverlayProps {
  open: boolean;
  onClose: () => void;
  /** 데스크톱 팝오버를 붙일 위치(열 때의 뷰포트 좌표) */
  anchor: OverlayAnchor | null;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  /** 모바일 시트 높이 — fit: 내용만큼(최대 85dvh) · half: 화면 절반쯤(위에 고른 문장이 보이게) */
  sheetSize?: "fit" | "half";
}

const POPOVER_WIDTH = 380;
const GAP = 8;
const MARGIN = 12;

/** CSS 변수로 잡힌 고정 바 높이(px). calc·env 도 실제 값으로 잰다. */
function insetPx(variable: string): number {
  const probe = document.createElement("div");
  probe.style.cssText = `position:absolute;visibility:hidden;height:var(${variable},0px)`;
  document.body.append(probe);
  const height = probe.offsetHeight;
  probe.remove();
  return height;
}

/**
 * 설명 본문 — 다 들어가면 그대로 보이고, 넘칠 때만 패널 안에서 스크롤한다. 넘치는 동안 아래쪽에 페이드와 '아래로 이어져요'를 보여
 * 잘린 건지 스크롤되는 건지 구분되게 하고, 끝까지 내리면 없앤다.
 */
function OverlayBody({ scrollRef, contentRef, labelId, children }: { scrollRef: RefObject<HTMLDivElement>; contentRef: RefObject<HTMLDivElement>; labelId: string; children: ReactNode }) {
  const t = useT(messages);
  const [more, setMore] = useState(false);
  // 탭 정지는 '넘치는가'로만 정한다. 끝까지 내렸다고 없애면 포커스가 body 로 빠져 Esc·Tab 이 먹지 않는다.
  const [overflowing, setOverflowing] = useState(false);

  useEffect(() => {
    const scroller = scrollRef.current;
    const content = contentRef.current;
    if (!scroller || !content) { return undefined; }
    const update = () => {
      setOverflowing(scroller.scrollHeight - scroller.clientHeight > 4);
      setMore(scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight > 4);
    };
    update();
    scroller.addEventListener("scroll", update, { passive: true });
    const observer = new ResizeObserver(update);
    observer.observe(scroller);
    observer.observe(content);
    return () => {
      scroller.removeEventListener("scroll", update);
      observer.disconnect();
    };
  }, [scrollRef, contentRef]);

  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      {/* 넘칠 때만 탭 정지 — 키보드로도 끝까지 읽을 수 있게 */}
      <div
        ref={scrollRef}
        tabIndex={overflowing ? 0 : undefined}
        role={overflowing ? "region" : undefined}
        aria-labelledby={overflowing ? labelId : undefined}
        className={cn("min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-4 pt-2", focusRing)}
      >
        <div ref={contentRef}>{children}</div>
      </div>
      {more && (
        <div aria-hidden="true" className="pointer-events-none absolute inset-x-0 bottom-0 flex h-[min(4rem,25%)] items-end justify-center bg-gradient-to-t from-canvas-elevated from-45% to-transparent pb-1.5">
          <span className="inline-flex items-center gap-1 text-xs font-medium text-fg-secondary">
            <ChevronDown className="size-3.5" />{t.continuesBelow}
          </span>
        </div>
      )}
    </div>
  );
}

/**
 * 강의 읽기 중 보조 설명 — 본문을 밀어내거나 스크롤 위치를 바꾸지 않는 오버레이.
 * 데스크톱(md+): 대상 옆에 붙는 팝오버(문서 좌표, 본문과 함께 스크롤). 모바일: 하단 시트.
 * 설명은 열린 뒤에 도착해 커지므로 내용 크기가 바뀔 때마다 자리를 다시 잡는다(한 번만 재면 화면 아래로 잘렸다, 2026-11-10).
 * BottomSheet 와 달리 body 스크롤 잠금(position:fixed)을 쓰지 않아 열고 닫아도 window.scrollY 가 그대로다.
 * 포커스 이동·복귀는 preventScroll 로만 한다(useFocusTrap).
 */
export function ReadingOverlay({ open, onClose, anchor, title, description, children, footer, sheetSize = "fit" }: ReadingOverlayProps) {
  const t = useT(messages);
  const desktop = useMediaQuery("(min-width: 768px)");
  const panelRef = useRef<HTMLDivElement>(null);
  const [mounted, setMounted] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [placement, setPlacement] = useState<OverlayPlacement | null>(null);
  const titleId = useId();

  useEffect(() => setMounted(true), []);
  useFocusTrap(panelRef, open && mounted, onClose);

  // 데스크톱: 화면 안에 다 들어가는 쪽에 붙인다. 문서 좌표로 바꿔 두어 스크롤해도 대상과 함께 움직인다.
  useLayoutEffect(() => {
    if (!open || !desktop || !anchor || !mounted) {
      setPlacement(null);
      return undefined;
    }
    const panel = panelRef.current;
    const scroller = scrollRef.current;
    const content = contentRef.current;
    if (!panel || !scroller || !content) { return undefined; }
    const openedAtY = window.scrollY;
    const place = () => {
      const shift = window.scrollY - openedAtY;
      const next = placeOverlay({
        anchor: { top: anchor.top - shift, bottom: anchor.bottom - shift, left: anchor.left },
        naturalHeight: panel.offsetHeight - scroller.clientHeight + scroller.scrollHeight,
        width: POPOVER_WIDTH, viewportW: document.documentElement.clientWidth, viewportH: window.innerHeight,
        topInset: insetPx("--app-header-offset"), bottomInset: insetPx("--app-bottom-offset"), gap: GAP, margin: MARGIN,
      });
      setPlacement({ ...next, top: next.top + window.scrollY, left: next.left + window.scrollX });
    };
    place();
    const observer = new ResizeObserver(place);
    observer.observe(content);
    observer.observe(panel);
    window.addEventListener("resize", place);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", place);
    };
  }, [open, desktop, anchor, mounted]);

  // 바깥을 누르면 닫는다(데스크톱 팝오버 — 시트는 배경 레이어가 받는다).
  useEffect(() => {
    if (!open || !desktop) { return; }
    function onPointerDown(event: PointerEvent) {
      if (!eventWithin(event, panelRef.current)) { onClose(); }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [open, desktop, onClose]);

  if (!open || !mounted) { return null; }

  const header = (
    <div className="flex shrink-0 items-start gap-2 px-4 pb-1 pt-3">
      <div className="min-w-0 flex-1">
        <h2 id={titleId} className="text-base font-semibold text-fg">{title}</h2>
        {description ? <p className="mt-0.5 text-xs text-fg-muted">{description}</p> : null}
      </div>
      <IconButton aria-label={t.close} onClick={onClose} className="-mr-2 -mt-1">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </IconButton>
    </div>
  );
  const body = <OverlayBody scrollRef={scrollRef} contentRef={contentRef} labelId={titleId}>{children}</OverlayBody>;
  const foot = footer ? <div className="flex shrink-0 justify-end gap-2 border-t border-line px-4 py-2">{footer}</div> : null;

  if (desktop) {
    return createPortal(
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="false"
        aria-labelledby={titleId}
        tabIndex={-1}
        style={{ top: placement?.top ?? -9999, left: placement?.left ?? -9999, maxHeight: placement?.maxHeight, width: POPOVER_WIDTH }}
        className={cn(
          "absolute z-panel flex flex-col overflow-hidden rounded-lg border border-line",
          "bg-canvas-elevated text-fg shadow-overlay outline-none animate-fade-in motion-reduce:animate-none",
        )}
      >
        {header}
        {body}
        {foot}
      </div>,
      portalRoot(),
    );
  }

  return createPortal(
    <div className="fixed inset-0 z-sheet flex items-end">
      {/* touch-none: 배경을 끌어도 뒤의 본문이 스크롤되지 않게(스크롤 잠금 없이) */}
      <div aria-hidden="true" className="absolute inset-0 touch-none bg-overlay animate-fade-in motion-reduce:animate-none" onClick={onClose} />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={cn(
          "relative flex w-full flex-col overflow-hidden rounded-t-xl border-t border-line",
          sheetSize === "half" ? "h-[55dvh]" : "max-h-[85dvh]",
          "bg-canvas-elevated text-fg shadow-overlay outline-none pb-safe-bottom animate-sheet-up motion-reduce:animate-none",
        )}
      >
        <div className="flex shrink-0 justify-center pt-2" aria-hidden="true">
          <span className="h-1 w-9 rounded-full bg-line-strong" />
        </div>
        {header}
        {body}
        {foot}
      </div>
    </div>,
    portalRoot(),
  );
}
